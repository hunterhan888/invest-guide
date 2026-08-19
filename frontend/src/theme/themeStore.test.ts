import { describe, it, expect, beforeEach } from 'vitest';
import { useThemeStore } from './themeStore';

describe('themeStore', () => {
  beforeEach(() => {
    localStorage.clear();
    useThemeStore.setState({ mode: 'light' });
  });

  it('toggle 在 light/dark 间切换并持久化', () => {
    useThemeStore.getState().setMode('light');
    useThemeStore.getState().toggle();
    expect(useThemeStore.getState().mode).toBe('dark');
    expect(localStorage.getItem('investguide.theme')).toBe('dark');
  });

  it('hydrate 读取已保存偏好', () => {
    localStorage.setItem('investguide.theme', 'dark');
    useThemeStore.getState().hydrate();
    expect(useThemeStore.getState().mode).toBe('dark');
  });
});
