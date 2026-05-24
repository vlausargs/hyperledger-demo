import { redirect } from '@sveltejs/kit';
import { getToken, parseJWT } from '$lib/server/auth';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies }) => {
  const token = getToken(cookies);
  if (!token) {
    throw redirect(302, '/login');
  }

  const claims = parseJWT(token);
  if (!claims || claims.exp * 1000 < Date.now()) {
    throw redirect(302, '/login');
  }

  return {
    user: {
      username: claims.sub,
      org: claims.org,
      role: claims.role
    }
  };
};
