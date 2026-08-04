import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore, TOKEN_KEY, SESSION_TOKEN_KEY } from './authStore';

describe('authStore', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    useAuthStore.getState().logout();
  });

  it('初始无 token', () => {
    expect(useAuthStore.getState().token).toBeNull();
    expect(useAuthStore.getState().user).toBeNull();
  });

  it('login 设置 token 并持久化到 localStorage', () => {
    useAuthStore.getState().login({
      token: 'tok_1',
      user: { id: '1', email: 'a@b.com', displayName: 'A' },
    });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('tok_1');
    expect(useAuthStore.getState().user?.email).toBe('a@b.com');
  });

  it('logout 清空 token 与 user', () => {
    useAuthStore.getState().login({
      token: 'tok_1',
      user: { id: '1', email: 'a@b.com', displayName: 'A' },
    });
    useAuthStore.getState().logout();
    expect(useAuthStore.getState().token).toBeNull();
    expect(useAuthStore.getState().user).toBeNull();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
  });

  it('hydrate 从 localStorage 恢复 token', () => {
    localStorage.setItem(TOKEN_KEY, 'tok_2');
    useAuthStore.getState().hydrate();
    expect(useAuthStore.getState().token).toBe('tok_2');
  });

  it('remember=false 时 token 持久化到 sessionStorage', () => {
    useAuthStore.getState().login({
      token: 'tok_session',
      user: { id: '2', email: 'b@b.com', displayName: 'B' },
      remember: false,
    });
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(sessionStorage.getItem(SESSION_TOKEN_KEY)).toBe('tok_session');
  });

  it('hydrate 优先 localStorage，缺失时读 sessionStorage', () => {
    sessionStorage.setItem(SESSION_TOKEN_KEY, 'tok_session_2');
    useAuthStore.getState().hydrate();
    expect(useAuthStore.getState().token).toBe('tok_session_2');

    localStorage.setItem(TOKEN_KEY, 'tok_local');
    useAuthStore.getState().hydrate();
    expect(useAuthStore.getState().token).toBe('tok_local');
  });

  it('logout 同时清除 localStorage 与 sessionStorage 中的 token', () => {
    useAuthStore.getState().login({
      token: 'tok_local',
      user: { id: '1', email: 'a@b.com', displayName: 'A' },
    });
    useAuthStore.getState().login({
      token: 'tok_session',
      user: { id: '2', email: 'b@b.com', displayName: 'B' },
      remember: false,
    });
    useAuthStore.getState().logout();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
    expect(sessionStorage.getItem(SESSION_TOKEN_KEY)).toBeNull();
  });
});
