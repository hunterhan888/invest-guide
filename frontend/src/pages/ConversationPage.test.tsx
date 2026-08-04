import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { SWRConfig } from 'swr';
import { __resetMocks } from '@/api/client';
import { installAuthMocks, __resetAuthData } from '@/api/mock/auth';
import { installConversationMocks, __resetConversationData } from '@/api/mock/conversation';
import { useAuthStore } from '@/stores/authStore';
import ConversationPage from './ConversationPage';

describe('ConversationPage', () => {
  beforeEach(() => {
    localStorage.clear();
    __resetMocks();
    installAuthMocks();
    installConversationMocks();
    __resetAuthData();
    __resetConversationData();
  });

  it('发送问题后流式渲染回答', async () => {
    const user = userEvent.setup();
    const { register } = await import('@/api/auth/auth');
    const { createConversation } = await import('@/api/conversation/conversation');
    const r = await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    useAuthStore.getState().login({ token: r.token, user: r.user });
    const conv = await createConversation({});

    render(
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        <MemoryRouter initialEntries={[`/conversations/${conv.id}`]}>
          <Routes>
            <Route path="/conversations/:id" element={<ConversationPage />} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>,
    );

    const userInput = await screen.findByRole('textbox');
    await user.type(userInput, '越南税收');
    await user.click(screen.getByRole('button', { name: /发\s*送/ }));

    await waitFor(
      () => {
        expect(screen.getByText(/针对「越南税收」/)).toBeInTheDocument();
      },
      { timeout: 10000 },
    );
  }, 20000);

  it('首页携带 pendingMessageId 到达时，自动开流渲染首个回答', async () => {
    const { register } = await import('@/api/auth/auth');
    const { createConversation, sendMessage } = await import('@/api/conversation/conversation');
    const r = await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    useAuthStore.getState().login({ token: r.token, user: r.user });
    const conv = await createConversation({});
    const { messageId } = await sendMessage(conv.id, { content: '越南税收' });

    render(
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        <MemoryRouter
          initialEntries={[
            { pathname: `/conversations/${conv.id}`, state: { pendingMessageId: messageId } },
          ]}
        >
          <Routes>
            <Route path="/conversations/:id" element={<ConversationPage />} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>,
    );

    await waitFor(
      () => {
        expect(screen.getByText(/针对「越南税收」/)).toBeInTheDocument();
      },
      { timeout: 10000 },
    );
  }, 20000);

  it('流式过程中 sources 事件渲染引用来源', async () => {
    const user = userEvent.setup();
    const { register } = await import('@/api/auth/auth');
    const { createConversation } = await import('@/api/conversation/conversation');
    const r = await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    useAuthStore.getState().login({ token: r.token, user: r.user });
    const conv = await createConversation({});

    render(
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        <MemoryRouter initialEntries={[`/conversations/${conv.id}`]}>
          <Routes>
            <Route path="/conversations/:id" element={<ConversationPage />} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>,
    );

    const userInput = await screen.findByRole('textbox');
    await user.type(userInput, '越南税收');
    await user.click(screen.getByRole('button', { name: /发\s*送/ }));

    // mock SSE 的 sources 事件携带 title；流式完成后来源应渲染
    await waitFor(
      () => {
        expect(screen.getAllByText(/对外投资合作国别指南 - 越南/).length).toBeGreaterThan(0);
      },
      { timeout: 10000 },
    );
  }, 20000);
});
