import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import LoginPage from './LoginPage';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<div>Register page</div>} />
        <Route path="/" element={<div>Home</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('LoginPage', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    useAuthStore.getState().logout();
    vi.clearAllMocks();
  });

  it('渲染邮箱、密码、记住我与登录按钮', () => {
    renderAt('/login');
    expect(screen.getByPlaceholderText(/邮箱/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/密码/i)).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: /记住我/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /登\s*录/ })).toBeInTheDocument();
  });

  it('提供注册链接并跳转 /register', async () => {
    renderAt('/login');
    await userEvent.click(screen.getByRole('button', { name: /注册/ }));
    expect(screen.getByText('Register page')).toBeInTheDocument();
  });

  it('成功登录后写入 token', async () => {
    const { installAuthMocks, __resetAuthData } = await import('@/api/mock/auth');
    const { __resetMocks } = await import('@/api/client');
    __resetMocks();
    installAuthMocks();
    __resetAuthData();
    const { register } = await import('@/api/auth/auth');
    await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });

    renderAt('/login');
    await userEvent.type(screen.getByPlaceholderText(/邮箱/i), 'a@b.com');
    await userEvent.type(screen.getByPlaceholderText(/密码/i), 'pass1234');
    await userEvent.click(screen.getByRole('button', { name: /登\s*录/ }));
    await waitFor(() => {
      expect(useAuthStore.getState().token).not.toBeNull();
    });
  });
});
