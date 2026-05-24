import { fail, redirect } from '@sveltejs/kit';
import { apiRequest, ApiError } from '$lib/server/api';
import { setToken } from '$lib/server/auth';
import type { Actions } from './$types';

export const actions: Actions = {
  default: async ({ request, cookies }) => {
    const data = await request.formData();
    const username = data.get('username') as string;
    const password = data.get('password') as string;

    try {
      const result = await apiRequest('/api/v1/auth/login', {
        method: 'POST',
        body: JSON.stringify({ username, password })
      });
      setToken(cookies, result.token);
    } catch (e) {
      if (e instanceof ApiError) {
        return fail(401, { error: e.message });
      }
      return fail(500, { error: 'Server error' });
    }

    throw redirect(303, '/dashboard');
  }
};
