// 入口：UI 事件 / 屏切换 / 受益人解密渲染
// 加密见 crypto.js，后端交互见 api.js。零知识：密钥与明文只在本地。
import {
  bufToB64, b64ToBuf, deriveKey, encryptText, decryptText, parseBlob, uuidV4,
} from './crypto.js';
import {
  API_BASE, apiPut, apiHeartbeat, apiGetTrigger, apiGetVault,
} from './api.js';

const $ = (s) => document.querySelector(s);

// ---------- Toast 通知（替代生硬 alert）----------
function toast(msg, kind = '', ms = 3200) {
  const root = $('#toast-root');
  const el = document.createElement('div');
  el.className = 'toast' + (kind ? ' ' + kind : '');
  el.textContent = msg;
  root.appendChild(el);
  setTimeout(() => {
    el.classList.add('fade');
    setTimeout(() => el.remove(), 320);
  }, ms);
}

// ---------- 本地存储（仅缓存 ID / salt / 最近密文，方便再次打开）----------
const LS = { id: 'fm_id', salt: 'fm_salt', vault: 'fm_vault_local', ben: 'fm_ben_local' };
const load = (k, d) => { const v = localStorage.getItem(k); return v ? JSON.parse(v) : d; };
const save = (k, v) => localStorage.setItem(k, JSON.stringify(v));

// 运行时状态（不落盘）
let masterKey = null;
let releaseKey = null;
let currentId = null;
let currentSalt = null;
let currentHb = null;   // 最近一次心跳时间（unix ms）
let lastTrigger = null; // 受益人侧最近一次触发状态，用于倒计时刷新

// ---------- 屏切换 ----------
function show(screen) {
  ['setup', 'vault', 'release'].forEach((s) => $('#screen-' + s).classList.add('hidden'));
  $('#screen-' + screen).classList.remove('hidden');
  window.scrollTo(0, 0);
}

// ---------- 工具 ----------
function fmtTime(ms) {
  if (!ms) return '—';
  return new Date(ms).toLocaleString('zh-CN');
}
function fmtDur(sec) {
  if (sec <= 0) return '0 秒';
  if (sec < 60) return sec + ' 秒';
  if (sec < 3600) return Math.floor(sec / 60) + ' 分' + (sec % 60) + ' 秒';
  return Math.floor(sec / 3600) + ' 小时' + Math.floor((sec % 3600) / 60) + ' 分';
}

// 已释放：禁用"加密存入"并显横幅
function setReleasedUI(released) {
  const banner = $('#released-banner');
  const btn = $('#btn-add');
  if (released) {
    banner.classList.remove('hidden');
    btn.disabled = true;
    btn.textContent = '已释放 · 不可修改';
  } else {
    banner.classList.add('hidden');
    btn.disabled = false;
    btn.textContent = '加密存入';
  }
}

// ---------- 创建 / 打开（主人）----------
$('#btn-create').addEventListener('click', async () => {
  const mp = $('#master-pwd').value;
  const rp = $('#release-pwd').value;
  if (!mp || !rp) { toast('主密码和释放密码都要填', 'warn'); return; }

  const providedId = $('#vault-id').value.trim();
  let id = providedId || load(LS.id, null);
  let salt = load(LS.salt, null);

  // —— 打开已有保险库 ——
  if (id) {
    let rec;
    try { rec = await apiGetVault(id); } catch (e) { toast(e.message, 'error'); return; }
    if (!rec) { toast('服务端找不到该保险库（ID 错误或后端未运行）', 'error'); return; }
    if (!salt) salt = rec.salt;
    masterKey = await deriveKey(mp, b64ToBuf(salt));
    releaseKey = await deriveKey(rp, b64ToBuf(salt));
    const vBlob = parseBlob(rec.vault);
    if (vBlob) {
      try { await decryptText(masterKey, vBlob.iv, vBlob.ct); }
      catch { toast('主密码不正确，无法打开该保险库', 'error'); masterKey = releaseKey = null; return; }
    }
    currentId = id; currentSalt = salt; currentHb = rec.heartbeat_at || Date.now();
    save(LS.id, id); save(LS.salt, salt);
    save(LS.vault, rec.vault || ''); save(LS.ben, rec.beneficiary || '');
    enterVault();
    return;
  }

  // —— 新建保险库 ——
  const email = $('#owner-email').value.trim();
  const benEmail = $('#ben-email').value.trim();
  if (!salt) salt = bufToB64(crypto.getRandomValues(new Uint8Array(16)));
  id = uuidV4();
  masterKey = await deriveKey(mp, b64ToBuf(salt));
  releaseKey = await deriveKey(rp, b64ToBuf(salt));
  try { await apiPut(id, salt, '', '', email, benEmail); }
  catch (e) { toast(e.message, 'error'); return; }
  currentId = id; currentSalt = salt; currentHb = Date.now();
  save(LS.id, id); save(LS.salt, salt); save(LS.vault, ''); save(LS.ben, '');
  enterVault();
  toast('保险库已创建，取用链接已就绪', 'ok');
});

// ---------- 主人视图 ----------
function enterVault() {
  show('vault');
  $('#vault-id-disp').textContent = currentId;
  renderHeartbeat();
  renderEntries();
  // 若已释放，直接进禁用态（避免误存）
  setReleasedUI(false);
}
function renderHeartbeat() { $('#hb-time').textContent = fmtTime(currentHb); }

$('#btn-alive').addEventListener('click', async () => {
  try {
    const v = await apiHeartbeat(currentId);
    currentHb = v.heartbeat_at || Date.now();
    renderHeartbeat();
    setReleasedUI(v.trigger_status === 'released');
    toast('已向服务端报到，释放已取消（如有）', 'ok');
  } catch (e) { toast(e.message, 'error'); }
});

$('#btn-lock').addEventListener('click', () => {
  masterKey = releaseKey = null;
  show('setup');
});

$('#btn-copy-link').addEventListener('click', async () => {
  const url = `${location.origin}${location.pathname}?id=${encodeURIComponent(currentId)}`;
  try { await navigator.clipboard.writeText(url); } catch { /* 忽略，手动复制 */ }
  toast('取用链接已复制：' + url, 'ok', 5000);
});

async function getAllEntries() {
  const blob = parseBlob(load(LS.vault, ''));
  if (!blob) return [];
  try {
    const json = await decryptText(masterKey, blob.iv, blob.ct);
    return JSON.parse(json);
  } catch { return []; }
}
async function renderEntries() {
  if (!masterKey) return;
  const list = await getAllEntries();
  const ul = $('#entry-list');
  ul.innerHTML = '';
  $('#entry-empty').classList.toggle('hidden', list.length > 0);
  for (const e of list) {
    const li = document.createElement('li');
    li.innerHTML = `<div class="t"></div><div class="b"></div>`;
    li.querySelector('.t').textContent = e.title;
    li.querySelector('.b').textContent = e.body;
    ul.appendChild(li);
  }
}

$('#btn-add').addEventListener('click', async () => {
  const title = $('#entry-title').value.trim();
  const body = $('#entry-body').value;
  if (!title) { toast('填个标题', 'warn'); return; }
  const list = await getAllEntries();
  list.push({ title, body, at: Date.now() });
  const vStr = JSON.stringify(await encryptText(masterKey, JSON.stringify(list)));
  const bStr = JSON.stringify(await encryptText(releaseKey, JSON.stringify(list)));
  const email = $('#owner-email').value.trim();
  const benEmail = $('#ben-email').value.trim();
  try {
    await apiPut(currentId, currentSalt, vStr, bStr, email, benEmail);
  } catch (e) {
    if (e.message === 'RELEASED') {
      setReleasedUI(true);
      toast('保险库已释放，不能再修改内容。受益人现在可取走。', 'warn', 4500);
      return;
    }
    toast(e.message, 'error');
    return;
  }
  save(LS.vault, vStr); save(LS.ben, bStr);
  $('#entry-title').value = '';
  $('#entry-body').value = '';
  renderEntries();
  toast('已加密存入', 'ok');
});

// ---------- 受益人视图 ----------
function setReleaseStatus(kind, text) {
  const box = $('#release-status');
  box.hidden = false;
  box.className = 'status-box' + (kind ? ' ' + kind : '');
  box.textContent = text;
}
function renderReleaseList(list) {
  const ul = $('#release-list');
  ul.innerHTML = '';
  if (!list.length) { ul.innerHTML = '<li class="empty"><div>包是空的。</div></li>'; return; }
  for (const e of list) {
    const li = document.createElement('li');
    li.innerHTML = `<div class="t"></div><div class="b"></div>`;
    li.querySelector('.t').textContent = e.title;
    li.querySelector('.b').textContent = e.body;
    ul.appendChild(li);
  }
}
$('#btn-ben-enter').addEventListener('click', () => {
  setReleaseStatus('', '输入保险库 ID 和释放密码，点"查询并解密取走"。');
  show('release');
});

$('#btn-release').addEventListener('click', async () => {
  const id = $('#ben-id').value.trim();
  const rp = $('#ben-pwd').value;
  if (!id || !rp) { toast('填保险库 ID 和释放密码', 'warn'); return; }

  let tr;
  try { tr = await apiGetTrigger(id); }
  catch (e) { toast(e.message, 'error'); return; }
  if (!tr) { setReleaseStatus('error', '找不到该保险库（ID 错误或后端未运行）。'); return; }
  lastTrigger = tr;

  if (tr.trigger_status !== 'released') {
    if (tr.trigger_status === 'grace') {
      const left = Math.max(0, Math.ceil((tr.grace_ends_at - Date.now()) / 1000));
      setReleaseStatus('grace', `⏳ 反悔窗口中，主人仍可取消；约剩 ${fmtDur(left)}。窗口过后即可取走。`);
    } else {
      setReleaseStatus('normal', '未触发：主人心跳正常，暂不可取。');
    }
    return;
  }

  // —— 已释放：取密文并解密 ——
  let rec;
  try { rec = await apiGetVault(id); }
  catch (e) { toast(e.message, 'error'); return; }
  if (!rec || !rec.beneficiary) { setReleaseStatus('error', '没有可取用的受益人包。'); return; }
  const key = await deriveKey(rp, b64ToBuf(rec.salt || currentSalt));
  try {
    const blob = parseBlob(rec.beneficiary);
    const json = await decryptText(key, blob.iv, blob.ct);
    const list = JSON.parse(json);
    renderReleaseList(list);
    setReleaseStatus('released', '✅ 已释放，内容如下（见下方）。');
    toast('解密成功', 'ok');
  } catch {
    setReleaseStatus('error', '释放密码不正确。');
  }
});

// 受益人侧倒计时刷新（仅本地文本，不再请求网络）
setInterval(() => {
  if ($('#screen-release').classList.contains('hidden')) return;
  if (!lastTrigger || lastTrigger.trigger_status !== 'grace') return;
  const left = Math.max(0, Math.ceil((lastTrigger.grace_ends_at - Date.now()) / 1000));
  setReleaseStatus('grace', `⏳ 反悔窗口中，主人仍可取消；约剩 ${fmtDur(left)}。窗口过后即可取走。`);
}, 1000);

// ---------- 启动 ----------
// 支持取用深链：?id=xxx 直接进受益人屏并填好 ID
const sp = new URLSearchParams(location.search);
const deepId = sp.get('id');
if (deepId) {
  $('#ben-id').value = deepId;
  setReleaseStatus('', '输入释放密码，点"查询并解密取走"。');
  show('release');
} else {
  show('setup');
}
console.log('[family-password] API_BASE =', API_BASE);
