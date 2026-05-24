import { json } from '@sveltejs/kit';
import { apiRequest, ApiError } from '$lib/server/api';
import { getToken } from '$lib/server/auth';
import type { RequestHandler } from './$types';

export const GET: RequestHandler = async ({ params, cookies }) => {
  const token = getToken(cookies);
  if (!token) return json({ error: 'unauthorized' }, { status: 401 });

  try {
    const result = await apiRequest(`/api/v1/products/${params.id}/provenance`, {}, token);
    return json(result);
  } catch (e) {
    if (e instanceof ApiError) {
      return json({ error: e.message }, { status: e.status });
    }
    return json({ error: 'Server error' }, { status: 500 });
  }
};
