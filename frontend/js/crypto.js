// 加密层（纯逻辑，无 DOM，可被浏览器与 node --test 复用）
// 零知识根基：主密码 / 释放密码 / 密钥 K 永远不出浏览器；服务端只见密文。

// base64 <-> ArrayBuffer
export function bufToB64(buf) {
  const bytes = new Uint8Array(buf);
  let bin = '';
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin);
}
export function b64ToBuf(b64) {
  const bin = atob(b64);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes.buffer;
}

// 密钥派生（PBKDF2, 150k 迭代, SHA-256）+ AES-GCM(256)
export async function deriveKey(password, saltBuf) {
  const enc = new TextEncoder();
  const base = await crypto.subtle.importKey('raw', enc.encode(password), 'PBKDF2', false, ['deriveKey']);
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt: saltBuf, iterations: 150000, hash: 'SHA-256' },
    base,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

// 加密：返回 { iv, ct }（base64）。调用方负责把它 JSON 化后上传或本地缓存。
export async function encryptText(key, text) {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const ct = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, new TextEncoder().encode(text));
  return { iv: bufToB64(iv), ct: bufToB64(ct) };
}

// 解密：ivB64 / ctB64 为 base64 字符串。密码错误会抛异常（调用方据此判断密码对错）。
export async function decryptText(key, ivB64, ctB64) {
  const pt = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: b64ToBuf(ivB64) }, key, b64ToBuf(ctB64));
  return new TextDecoder().decode(pt);
}

// 密文在存储 / 上传时是 JSON 字符串 "{\"iv\":...,\"ct\":...}"，统一解析为对象；空或非法返回 null。
export function parseBlob(str) {
  if (!str) return null;
  try { return JSON.parse(str); } catch { return null; }
}

// 生成 UUID v4。优先用原生 crypto.randomUUID（仅安全上下文可用：HTTPS 或 localhost）；
// 局域网明文 HTTP 不是安全上下文，crypto.randomUUID 为 undefined，需兜底实现。
export function uuidV4() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  const bytes = crypto.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
  bytes[8] = (bytes[8] & 0x3f) | 0x80; // variant 10xx
  const hex = [...bytes].map(b => b.toString(16).padStart(2, '0'));
  return (
    hex.slice(0, 4).join('') + '-' +
    hex.slice(4, 6).join('') + '-' +
    hex.slice(6, 8).join('') + '-' +
    hex.slice(8, 10).join('') + '-' +
    hex.slice(10, 16).join('')
  );
}
