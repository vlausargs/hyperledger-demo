import type { Cookies } from '@sveltejs/kit';

const TOKEN_COOKIE = 'hlf_token';

export function setToken(cookies: Cookies, token: string) {
  cookies.set(TOKEN_COOKIE, token, {
    path: '/',
    httpOnly: true,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'strict',
    maxAge: 60 * 15 // 15 min
  });
}

export function getToken(cookies: Cookies): string | undefined {
  return cookies.get(TOKEN_COOKIE);
}

export function clearToken(cookies: Cookies) {
  cookies.delete(TOKEN_COOKIE, { path: '/' });
}

export function parseJWT(token: string): { sub: string; org: string; role: string; exp: number } | null {
  try {
    const payload = token.split('.')[1];
    return JSON.parse(atob(payload));
  } catch {
    return null;
  }
}
