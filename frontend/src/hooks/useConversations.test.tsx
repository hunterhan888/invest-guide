import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { SWRConfig } from 'swr';
import { __resetMocks } from '@/api/client';
import { installConversationMocks, __resetConversationData } from '@/api/mock/conversation';
import { installAuthMocks, __resetAuthData } from '@/api/mock/auth';
import { useAuthStore } from '@/stores/authStore';
import { useConversations } from './useConversations';

describe('useConversations', () => {
  beforeEach(() => {
    localStorage.clear();
    __resetMocks();
    installAuthMocks();
    installConversationMocks();
    __resetAuthData();
    __resetConversationData();
  });

  it('未登录时不请求', async () => {
    useAuthStore.getState().logout();
    const { result } = renderHook(() => useConversations(), {
      wrapper: ({ children }) => (
        <SWRConfig value={{ provider: () => new Map() }}>{children}</SWRConfig>
      ),
    });
    expect(result.current.data).toBeUndefined();
  });

  it('登录后返回会话列表', async () => {
    const { register } = await import('@/api/auth/auth');
    const r = await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    useAuthStore.getState().login({ token: r.token, user: r.user });
    const { createConversation } = await import('@/api/conversation/conversation');
    await createConversation({ title: '首问' });

    const { result } = renderHook(() => useConversations(), {
      wrapper: ({ children }) => (
        <SWRConfig value={{ provider: () => new Map() }}>{children}</SWRConfig>
      ),
    });
    await waitFor(() => expect(result.current.data?.items.length).toBe(1));
  });
});
