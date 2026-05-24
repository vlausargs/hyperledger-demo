import { apiRequest } from '$lib/server/api';
import { getToken } from '$lib/server/auth';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ cookies }) => {
  const token = getToken(cookies)!;

  try {
    const products = await apiRequest('/api/v1/products?pageSize=5', {}, token);
    const shipments = await apiRequest('/api/v1/shipments?pageSize=5', {}, token);
    return { products, shipments };
  } catch {
    return { products: { products: [], count: 0 }, shipments: { shipments: [], count: 0 } };
  }
};
