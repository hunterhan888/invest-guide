import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import { act } from 'react';
import { ThemeProvider } from './ThemeProvider';
import { useThemeStore } from './themeStore';

describe('ThemeProvider', () => {
  beforeEach(() => {
    localStorage.clear();
    useThemeStore.setState({ mode: 'light' });
    document.body.removeAttribute('data-ds-dark-theme');
  });

  it('挂载后按 store 模式写入 body 属性', () => {
    render(
      <ThemeProvider>
        <div>child</div>
      </ThemeProvider>,
    );
    expect(document.body.hasAttribute('data-ds-dark-theme')).toBe(false);
  });

  it('store 切到 dark 后 body 带 data-ds-dark-theme', () => {
    render(
      <ThemeProvider>
        <div>child</div>
      </ThemeProvider>,
    );
    act(() => {
      useThemeStore.getState().setMode('dark');
    });
    expect(document.body.hasAttribute('data-ds-dark-theme')).toBe(true);
  });
});
