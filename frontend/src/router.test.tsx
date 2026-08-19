import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Navigate, Outlet, Route, Routes } from 'react-router-dom';
import { ToastProvider } from '@/primitives/ToastProvider';
import { useAuthStore } from '@/stores/authStore';
import HomePage from './pages/HomePage';
import LoginPage from './pages/LoginPage';
import RegisterPage from './pages/RegisterPage';

describe('RequireAuth', () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
    useAuthStore.getState().logout();
  });

  it('未登录访问 / 重定向到 /login', async () => {
    render(
      <ToastProvider>
        <MemoryRouter initialEntries={['/']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<RegisterPage />} />
            <Route path="/" element={<NavigateIfNoToken />}>
              <Route index element={<HomePage />} />
            </Route>
          </Routes>
        </MemoryRouter>
      </ToastProvider>,
    );
    expect(await screen.findByRole('button', { name: /登\s*录/ })).toBeInTheDocument();
  });

  it('已登录访问 / 渲染 HomePage', () => {
    useAuthStore.getState().login({
      token: 'tok_1',
      user: { id: '1', email: 'a@b.com', displayName: 'A' },
    });
    render(
      <ToastProvider>
        <MemoryRouter initialEntries={['/']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/" element={<NavigateIfNoToken />}>
              <Route index element={<HomePage />} />
            </Route>
          </Routes>
        </MemoryRouter>
      </ToastProvider>,
    );
    expect(screen.getByText(/你好，有什么可以帮你/)).toBeInTheDocument();
  });
});

function NavigateIfNoToken() {
  const token = useAuthStore((s) => s.token);
  if (!token) return <Navigate to="/login" replace />;
  return <Outlet />;
}
