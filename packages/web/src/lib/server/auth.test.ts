import { describe, it, expect, vi } from 'vitest';
import { setToken, getToken, clearToken, parseJWT } from './auth';
import type { Cookies } from '@sveltejs/kit';

function mockCookies(): Cookies & {
  store: Map<string, string>;
} {
  const store = new Map<string, string>();
  return {
    store,
    get: vi.fn((k: string) => store.get(k)),
    getAll: vi.fn(() =>
      Array.from(store.entries()).map(([name, value]) => ({ name, value }))
    ),
    set: vi.fn((k: string, v: string) => {
      store.set(k, v);
    }),
    delete: vi.fn((k: string) => {
      store.delete(k);
    }),
    serialize: vi.fn(() => '')
  } as unknown as Cookies & { store: Map<string, string> };
}

describe('auth token cookies', () => {
  it('setToken writes httpOnly cookie with 15-minute maxAge', () => {
    const cookies = mockCookies();
    setToken(cookies, 'abc123');
    expect(cookies.set).toHaveBeenCalledTimes(1);
    const [name, value, opts] = (cookies.set as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(name).toBe('hlf_token');
    expect(value).toBe('abc123');
    expect(opts.httpOnly).toBe(true);
    expect(opts.sameSite).toBe('strict');
    expect(opts.maxAge).toBe(60 * 15);
    expect(opts.path).toBe('/');
  });

  it('getToken reads what setToken stored', () => {
    const cookies = mockCookies();
    setToken(cookies, 'xyz');
    expect(getToken(cookies)).toBe('xyz');
  });

  it('getToken returns undefined when not set', () => {
    const cookies = mockCookies();
    expect(getToken(cookies)).toBeUndefined();
  });

  it('clearToken deletes the cookie at root path', () => {
    const cookies = mockCookies();
    setToken(cookies, 'gone');
    clearToken(cookies);
    expect(cookies.delete).toHaveBeenCalledWith('hlf_token', { path: '/' });
    expect(getToken(cookies)).toBeUndefined();
  });
});

describe('parseJWT', () => {
  it('decodes valid JWT payload', () => {
    // Header.Payload.Signature — only payload matters for parseJWT.
    const payload = { sub: 'admin', org: 'Org1MSP', role: 'admin', exp: 9999999999 };
    const b64 = btoa(JSON.stringify(payload));
    const token = `header.${b64}.sig`;
    const decoded = parseJWT(token);
    expect(decoded).toEqual(payload);
  });

  it('returns null for malformed input', () => {
    expect(parseJWT('not-a-jwt')).toBeNull();
    expect(parseJWT('a.b')).toBeNull();
    expect(parseJWT('a.not!base64.c')).toBeNull();
  });

  it('returns null for non-JSON payload', () => {
    const token = `h.${btoa('not-json')}.s`;
    expect(parseJWT(token)).toBeNull();
  });
});
