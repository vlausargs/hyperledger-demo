import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { apiRequest, ApiError } from './api';

describe('apiRequest', () => {
  const originalFetch = globalThis.fetch;

  afterEach(() => {
    globalThis.fetch = originalFetch;
    vi.restoreAllMocks();
  });

  it('returns JSON body on 2xx', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({ id: 'P-1', name: 'Widget' })
    } as Response);
    globalThis.fetch = mockFetch as unknown as typeof fetch;

    const data = await apiRequest('/api/v1/products/P-1');
    expect(data).toEqual({ id: 'P-1', name: 'Widget' });
    expect(mockFetch).toHaveBeenCalledTimes(1);

    const [url, opts] = mockFetch.mock.calls[0];
    expect(url).toContain('/api/v1/products/P-1');
    const headers = (opts as RequestInit).headers as Record<string, string>;
    expect(headers['Content-Type']).toBe('application/json');
    expect(headers['Authorization']).toBeUndefined();
  });

  it('adds Bearer Authorization when token provided', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => ({})
    } as Response);
    globalThis.fetch = mockFetch as unknown as typeof fetch;

    await apiRequest('/api/v1/products', {}, 'jwt-token');

    const [, opts] = mockFetch.mock.calls[0];
    const headers = (opts as RequestInit).headers as Record<string, string>;
    expect(headers['Authorization']).toBe('Bearer jwt-token');
  });

  it('throws ApiError with status + message on non-2xx with JSON body', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 404,
      statusText: 'Not Found',
      json: async () => ({ error: 'product not found' })
    } as Response) as unknown as typeof fetch;

    await expect(apiRequest('/api/v1/products/missing')).rejects.toMatchObject({
      status: 404,
      message: 'product not found'
    });
  });

  it('falls back to statusText when error body lacks .error field', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      statusText: 'Internal Server Error',
      json: async () => {
        throw new Error('non-json');
      }
    } as unknown as Response) as unknown as typeof fetch;

    try {
      await apiRequest('/api/v1/boom');
      expect.fail('expected throw');
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(500);
      expect((e as ApiError).message).toBe('Internal Server Error');
    }
  });

  it('forwards method and body for POST', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({ id: 'P-2' })
    } as Response);
    globalThis.fetch = mockFetch as unknown as typeof fetch;

    await apiRequest(
      '/api/v1/products',
      { method: 'POST', body: JSON.stringify({ id: 'P-2' }) },
      'tkn'
    );

    const [, opts] = mockFetch.mock.calls[0];
    const init = opts as RequestInit;
    expect(init.method).toBe('POST');
    expect(init.body).toBe(JSON.stringify({ id: 'P-2' }));
  });
});

describe('ApiError', () => {
  it('exposes status field and standard Error message', () => {
    const err = new ApiError(429, 'too many requests');
    expect(err).toBeInstanceOf(Error);
    expect(err.status).toBe(429);
    expect(err.message).toBe('too many requests');
  });
});
