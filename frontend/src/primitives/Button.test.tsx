import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';

describe('Button', () => {
  it('渲染 primary 按钮并触发 onClick', async () => {
    const onClick = vi.fn();
    const user = userEvent.setup();
    render(
      <Button variant="primary" onClick={onClick}>
        发送
      </Button>,
    );
    await user.click(screen.getByRole('button', { name: '发送' }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('loading 时禁用', () => {
    render(<Button loading>发送</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('disabled 禁用', () => {
    render(<Button disabled>发送</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('渲染文本 children', () => {
    render(<Button>文字</Button>);
    expect(screen.getByRole('button')).toHaveTextContent('文字');
  });
});
