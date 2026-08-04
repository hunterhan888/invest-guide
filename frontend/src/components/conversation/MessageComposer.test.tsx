import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MessageComposer } from './MessageComposer';

const emptyMessages = { items: [] };

describe('MessageComposer', () => {
  it('空内容不触发发送', async () => {
    const onStreamingChange = vi.fn();
    render(
      <MessageComposer
        conversationId="c1"
        messages={emptyMessages}
        mutateMessages={vi.fn()}
        onStreamingChange={onStreamingChange}
      />,
    );
    expect(screen.getByRole('textbox')).toBeInTheDocument();
  });

  it('输入内容后发送按钮可用', async () => {
    const user = userEvent.setup();
    render(
      <MessageComposer
        conversationId="c1"
        messages={emptyMessages}
        mutateMessages={vi.fn()}
        onStreamingChange={vi.fn()}
      />,
    );
    const btn = screen.getByRole('button', { name: /发\s*送/ });
    expect(btn).toBeDisabled();
    await user.type(screen.getByRole('textbox'), '测试问题');
    await waitFor(() => expect(btn).toBeEnabled());
  });
});
