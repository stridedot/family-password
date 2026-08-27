// 后端 API 封装（纯逻辑，无 DOM）
// API_BASE 读取优先级：URL ?api= 参数 > 默认值（生产后端地址）。
// 本地调试：在地址栏加 ?api=http://localhost:8080 即可切到本地后端。
// 说明：这里不用 import.meta.env（Vite 专属），因为前端是纯静态部署，浏览器里 import.meta.env 是 undefined。
const params = new URLSearchParams(location.search);
export const API_BASE = (
  params.get('api') || 'https://family-password.onrender.com'
).replace(/\/+$/, '');

// POST /api/vault —— 创建或更新保险库（密文上送服务端）
// email / beneficiary_email 为可选通知邮箱，仅非空时发送（更新时留空不会清空服务端已有邮箱）
export async function apiPut(id, salt, vault, beneficiary, email, beneficiaryEmail) {
  const body = { id, salt, vault, beneficiary };
  if (email) body.email = email;
  if (beneficiaryEmail) body.beneficiary_email = beneficiaryEmail;
  const res = await fetch(`${API_BASE}/api/vault`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    // 422：保险库已释放，拒绝写入（前端据此提示受益人可取走）
    if (res.status === 422) throw new Error('RELEASED');
    throw new Error('上传失败：' + res.status);
  }
  return res.json();
}

// POST /api/vault/:id/heartbeat —— 主人报到（刷新心跳 + 取消释放）
export async function apiHeartbeat(id) {
  const res = await fetch(`${API_BASE}/api/vault/${id}/heartbeat`, { method: 'POST' });
  if (!res.ok) throw new Error('心跳失败：' + res.status);
  return res.json();
}

// POST /api/vault/:id/trigger —— 查触发状态（服务端实时推进）
// 404 → 返回 null（保险库不存在）
export async function apiGetTrigger(id) {
  const res = await fetch(`${API_BASE}/api/vault/${id}/trigger`, { method: 'POST' });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error('查询触发状态失败：' + res.status);
  return res.json();
}

// POST /api/vault/:id —— 取完整记录（含密文）
// 404 → 返回 null
export async function apiGetVault(id) {
  const res = await fetch(`${API_BASE}/api/vault/${id}`, { method: 'POST' });
  if (res.status === 404) return null;
  if (!res.ok) throw new Error('获取保险库失败：' + res.status);
  return res.json();
}
