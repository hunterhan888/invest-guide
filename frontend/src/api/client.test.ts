import { describe, it, expect, beforeEach } from 'vitest';
import { request, registerMock, __resetMocks } from './client';
import { ApiError } from './types';

describe('request', () => {
  beforeEach(() => __resetMocks());

  it('成功解析 data', async () => {
    registerMock({
      match: () => true,
      handle: async () => ({ status: 200, body: { success: true, data: { id: '1' } } }),
    });
    const res = await request<{ id: string }>('GET', '/api/v1/x');
    expect(res).toEqual({ id: '1' });
  });

  it('success=false 抛 ApiError', async () => {
    registerMock({
      match: () => true,
      handle: async () => ({
        status: 401,
        body: { success: false, error: 'bad', code: 'UNAUTHORIZED' },
      }),
    });
    await expect(request('GET', '/api/v1/x')).rejects.toMatchObject({
      status: 401,
      code: 'UNAUTHORIZED',
    });
    expect(ApiError).toBeDefined();
  });

  it('未匹配路由抛 404', async () => {
    await expect(request('GET', '/api/v1/none')).rejects.toMatchObject({ status: 404 });
  });
});
