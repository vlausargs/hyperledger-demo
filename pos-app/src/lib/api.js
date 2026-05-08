const BASE = '/api/v1';

function authHeader() {
  const token = localStorage.getItem('token');
  return token ? { Authorization: `Bearer ${token}` } : {};
}

async function request(method, path, body) {
  const opts = {
    method,
    headers: { 'Content-Type': 'application/json', ...authHeader() }
  };
  if (body !== undefined) opts.body = JSON.stringify(body);
  const res = await fetch(BASE + path, opts);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || `HTTP ${res.status}`);
  return data;
}

export const api = {
  login: (username, password) =>
    request('POST', '/auth/login', { username, password }),

  verifyProduct: (id) =>
    request('GET', `/pos/verify/${encodeURIComponent(id)}`),

  getInventory: (ownerMsp = 'Org3MSP') =>
    request('GET', `/pos/inventory?ownerMsp=${ownerMsp}`),

  createSale: (payload) =>
    request('POST', '/pos/sales', payload),

  getSales: (params = {}) => {
    const qs = new URLSearchParams(params).toString();
    return request('GET', `/pos/sales${qs ? '?' + qs : ''}`);
  },

  getSale: (id) =>
    request('GET', `/pos/sales/${encodeURIComponent(id)}`)
};
