import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input } from './Input';
import { Textarea } from './Textarea';

describe('Input', () => {
  it('渲染输入框并支持受控值', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<Input value="" onChange={onChange} placeholder="邮箱" />);
    const input = screen.getByPlaceholderText('邮箱');
    await user.type(input, 'a@b.com');
    expect(onChange).toHaveBeenCalled();
  });

  it('支持 type=password', () => {
    render(<Input type="password" placeholder="密码" />);
    expect(screen.getByPlaceholderText('密码')).toHaveAttribute('type', 'password');
  });
});

describe('Textarea', () => {
  it('渲染 textarea', () => {
    render(<Textarea placeholder="问题" />);
    expect(screen.getByPlaceholderText('问题').tagName).toBe('TEXTAREA');
  });
});
