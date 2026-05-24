import { redirect } from '@sveltejs/kit';
import { clearToken } from '$lib/server/auth';
import type { Actions } from './$types';

export const actions: Actions = {
  default: async ({ cookies }) => {
    clearToken(cookies);
    throw redirect(303, '/login');
  }
};
