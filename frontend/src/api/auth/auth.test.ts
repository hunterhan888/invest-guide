import { describe, it, expect, beforeEach } from 'vitest';
import { __resetMocks } from '../client';
import { installAuthMocks, __resetAuthData } from '../mock/auth';
import { login, register } from './auth';

describe('auth api', () => {
  beforeEach(() => {
    localStorage.clear();
    __resetMocks();
    installAuthMocks();
    __resetAuthData();
  });

  it('login 成功返回 token+user', async () => {
    await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    const r = await login({ email: 'a@b.com', password: 'pass1234' });
    expect(r.token).toBeTypeOf('string');
    expect(r.user.email).toBe('a@b.com');
  });

  it('login 错误凭证抛 401', async () => {
    await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    await expect(login({ email: 'a@b.com', password: 'wrong' })).rejects.toMatchObject({
      status: 401,
      code: 'UNAUTHORIZED',
    });
  });

  it('register 重复邮箱抛 409', async () => {
    await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    await expect(
      register({ email: 'a@b.com', password: 'x', displayName: 'B' }),
    ).rejects.toMatchObject({
      status: 409,
      code: 'CONFLICT',
    });
  });
});
