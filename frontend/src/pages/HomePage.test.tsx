import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { App as AntdApp } from 'antd';
import HomePage from './HomePage';

const navigateMock = vi.fn();

vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return { ...actual, useNavigate: () => navigateMock };
});

vi.mock('@/api/conversation/conversation', () => ({
  createConversation: vi.fn(),
  sendMessage: vi.fn(),
}));

import { createConversation, sendMessage } from '@/api/conversation/conversation';

describe('HomePage', () => {
  beforeEach(() => {
    navigateMock.mockReset();
    vi.mocked(createConversation).mockReset();
    vi.mocked(sendMessage).mockReset();
  });

  it('从首页提问时携带 pendingMessageId 跳转到会话页', async () => {
    const user = userEvent.setup();
    vi.mocked(createConversation).mockResolvedValue({ id: 'conv-1' } as never);
    vi.mocked(sendMessage).mockResolvedValue({ messageId: 'msg-1' } as never);

    render(
      <AntdApp>
        <MemoryRouter>
          <HomePage />
        </MemoryRouter>
      </AntdApp>,
    );

    await user.type(screen.getByRole('textbox'), '我想去巴基斯坦投资');
    await user.click(screen.getByRole('button'));

    await vi.waitFor(() => {
      expect(createConversation).toHaveBeenCalledTimes(1);
    });
    expect(sendMessage).toHaveBeenCalledWith('conv-1', { content: '我想去巴基斯坦投资' });
    expect(navigateMock).toHaveBeenCalledWith('/conversations/conv-1', {
      state: { pendingMessageId: 'msg-1' },
    });
  });
});
