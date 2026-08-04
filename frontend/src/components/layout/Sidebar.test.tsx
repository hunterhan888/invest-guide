import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { SWRConfig } from 'swr';
import { __resetMocks } from '@/api/client';
import { installConversationMocks, __resetConversationData } from '@/api/mock/conversation';
import { installAuthMocks, __resetAuthData } from '@/api/mock/auth';
import { useAuthStore } from '@/stores/authStore';
import * as conversationApi from '@/api/conversation/conversation';
import Sidebar from './Sidebar';

async function login() {
  const { register } = await import('@/api/auth/auth');
  const r = await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
  useAuthStore.getState().login({ token: r.token, user: r.user });
  return r;
}

function renderSidebar() {
  return render(
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
      <MemoryRouter initialEntries={['/conversations/abc']}>
        <Routes>
          <Route path="/" element={<div>Sidebar home</div>} />
          <Route path="/conversations/:id" element={<Sidebar />} />
        </Routes>
      </MemoryRouter>
    </SWRConfig>,
  );
}

describe('Sidebar', () => {
  beforeEach(() => {
    localStorage.clear();
    __resetMocks();
    installAuthMocks();
    installConversationMocks();
    __resetAuthData();
    __resetConversationData();
  });

  it('点击新建对话不创建会话，回到首页', async () => {
    await login();
    const createSpy = vi.spyOn(conversationApi, 'createConversation').mockResolvedValue({
      id: 'x',
      title: '',
      country: null,
      createdAt: '',
      updatedAt: '',
    });
    renderSidebar();
    await userEvent.click(screen.getByRole('button', { name: /新建对话/ }));
    await waitFor(() => {
      expect(screen.getByText('Sidebar home')).toBeInTheDocument();
    });
    expect(createSpy).not.toHaveBeenCalled();
    createSpy.mockRestore();
  });
});
