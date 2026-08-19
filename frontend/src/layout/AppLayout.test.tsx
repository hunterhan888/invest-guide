import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { SWRConfig } from 'swr';
import AppLayout from './AppLayout';
import { useUiStore } from '@/stores/uiStore';

describe('AppLayout', () => {
  beforeEach(() => {
    localStorage.clear();
    useUiStore.getState().setCollapsed(false);
  });

  it('折叠按钮触发 sidebar 状态切换', async () => {
    const user = userEvent.setup();
    render(
      <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
        <MemoryRouter initialEntries={['/']}>
          <Routes>
            <Route path="/" element={<AppLayout />} />
          </Routes>
        </MemoryRouter>
      </SWRConfig>,
    );
    await user.click(screen.getByRole('button', { name: /折叠侧边栏|Toggle sidebar/ }));
    expect(useUiStore.getState().sidebarCollapsed).toBe(true);
  });
});
