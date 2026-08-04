import { describe, it, expect, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { __resetMocks } from '@/api/client';
import { installConversationMocks, __resetConversationData } from '@/api/mock/conversation';
import { installAuthMocks, __resetAuthData } from '@/api/mock/auth';
import { useAuthStore } from '@/stores/authStore';
import { useSSEStream, type SSEEvent } from './useSSEStream';
import { createConversation, sendMessage } from '@/api/conversation/conversation';

describe('useSSEStream', () => {
  beforeEach(() => {
    localStorage.clear();
    __resetMocks();
    installAuthMocks();
    installConversationMocks();
    __resetAuthData();
    __resetConversationData();
  });

  async function setup(forceError = false) {
    const { register } = await import('@/api/auth/auth');
    const r = await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    useAuthStore.getState().login({ token: r.token, user: r.user });
    const conv = await createConversation({});
    const { messageId } = await sendMessage(conv.id, { content: '测试问题' });
    const msgId = forceError ? 'force-error' : messageId;
    return { convId: conv.id, messageId: msgId };
  }

  it('正常流：sources → message* → done', async () => {
    const { convId, messageId } = await setup();
    const events: SSEEvent[] = [];
    renderHook(() =>
      useSSEStream({ convId, messageId, enabled: true, onEvent: (e) => events.push(e) }),
    );
    await waitFor(() => expect(events.some((e) => e.type === 'done')).toBe(true), {
      timeout: 5000,
    });
    expect(events[0]?.type).toBe('sources');
    expect(events.filter((e) => e.type === 'message').length).toBeGreaterThan(0);
  }, 15000);

  it('error 事件终止连接，无 done', async () => {
    const { convId, messageId } = await setup(true);
    const events: SSEEvent[] = [];
    renderHook(() =>
      useSSEStream({ convId, messageId, enabled: true, onEvent: (e) => events.push(e) }),
    );
    await waitFor(() => expect(events.some((e) => e.type === 'error')).toBe(true), {
      timeout: 5000,
    });
    expect(events.some((e) => e.type === 'done')).toBe(false);
  }, 15000);

  it('disabled 时不发起流', () => {
    const { convId, messageId } = { convId: 'c1', messageId: 'm1' };
    const events: SSEEvent[] = [];
    renderHook(() =>
      useSSEStream({ convId, messageId, enabled: false, onEvent: (e) => events.push(e) }),
    );
    expect(events).toHaveLength(0);
  });
});
