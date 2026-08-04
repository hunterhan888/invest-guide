import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import RegisterPage from './RegisterPage';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/login" element={<div>Login page</div>} />
        <Route path="/" element={<div>Home</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('RegisterPage', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    useAuthStore.getState().logout();
    vi.clearAllMocks();
  });

  it('渲染昵称、邮箱、密码与确认密码字段', () => {
    renderAt('/register');
    expect(screen.getByPlaceholderText(/昵称/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/邮箱/i)).toBeInTheDocument();
    expect(screen.getByPlaceholderText('密码')).toBeInTheDocument();
    expect(screen.getByPlaceholderText(/确认密码/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /注\s*册/ })).toBeInTheDocument();
  });

  it('两次密码不一致时提示错误', async () => {
    renderAt('/register');
    await userEvent.type(screen.getByPlaceholderText(/昵称/i), 'A');
    await userEvent.type(screen.getByPlaceholderText(/邮箱/i), 'a@b.com');
    await userEvent.type(screen.getByPlaceholderText('密码'), 'pass1234');
    await userEvent.type(screen.getByPlaceholderText(/确认密码/i), 'pass5678');
    await userEvent.click(screen.getByRole('button', { name: /注\s*册/ }));
    expect(await screen.findByText(/密码不一致/)).toBeInTheDocument();
  });

  it('提供登录链接并跳转 /login', async () => {
    renderAt('/register');
    await userEvent.click(screen.getByRole('button', { name: /登录/ }));
    expect(screen.getByText('Login page')).toBeInTheDocument();
  });

  it('成功注册后写入 token', async () => {
    const { installAuthMocks, __resetAuthData } = await import('@/api/mock/auth');
    const { __resetMocks } = await import('@/api/client');
    __resetMocks();
    installAuthMocks();
    __resetAuthData();

    renderAt('/register');
    await userEvent.type(screen.getByPlaceholderText(/昵称/i), 'A');
    await userEvent.type(screen.getByPlaceholderText(/邮箱/i), 'a@b.com');
    await userEvent.type(screen.getByPlaceholderText('密码'), 'pass1234');
    await userEvent.type(screen.getByPlaceholderText(/确认密码/i), 'pass1234');
    await userEvent.click(screen.getByRole('button', { name: /注\s*册/ }));
    await waitFor(() => {
      expect(useAuthStore.getState().token).not.toBeNull();
    });
  });
});
