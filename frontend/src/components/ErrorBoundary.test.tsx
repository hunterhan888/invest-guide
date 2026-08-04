import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ErrorBoundary } from './ErrorBoundary';

const Bomb = () => {
  throw new Error('boom');
};

describe('ErrorBoundary', () => {
  it('捕获渲染错误后显示回退 UI', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    render(
      <ErrorBoundary>
        <Bomb />
      </ErrorBoundary>,
    );
    expect(screen.getByText(/出错了/i)).toBeInTheDocument();
    spy.mockRestore();
  });
});
