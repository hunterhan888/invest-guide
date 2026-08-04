import { describe, it, expect, beforeEach } from 'vitest';
import { useUiStore } from './uiStore';

describe('uiStore', () => {
  beforeEach(() => localStorage.clear());

  it('toggleSidebar 翻转并持久化', () => {
    useUiStore.getState().setCollapsed(false);
    useUiStore.getState().toggleSidebar();
    expect(useUiStore.getState().sidebarCollapsed).toBe(true);
    expect(localStorage.getItem('investguide.sidebarCollapsed')).toBe('true');
  });
});
