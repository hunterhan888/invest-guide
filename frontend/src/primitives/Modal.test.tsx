import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from './Modal';

describe('Modal', () => {
  it('open 为 false 时不渲染', () => {
    render(<Modal open={false}>内容</Modal>);
    expect(screen.queryByText('内容')).not.toBeInTheDocument();
  });

  it('open 为 true 时渲染标题与内容', () => {
    render(
      <Modal open title="确认删除" onClose={vi.fn()}>
        内容
      </Modal>,
    );
    expect(screen.getByText('确认删除')).toBeInTheDocument();
    expect(screen.getByText('内容')).toBeInTheDocument();
  });

  it('点击遮罩触发 onClose', async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(
      <Modal open onClose={onClose}>
        内容
      </Modal>,
    );
    await user.click(screen.getByTestId('modal-mask'));
    expect(onClose).toHaveBeenCalled();
  });

  it('按 Esc 触发 onClose', async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(
      <Modal open onClose={onClose}>
        内容
      </Modal>,
    );
    await user.keyboard('{Escape}');
    expect(onClose).toHaveBeenCalled();
  });
});
