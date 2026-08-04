import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import MessageList from './MessageList';
import type { Message } from '@/api/conversation/types';

const msgs: Message[] = [
  {
    id: '1',
    role: 'user',
    content: '问题',
    sources: null,
    tokensUsed: null,
    createdAt: '2026-01-01T00:00:00Z',
  },
  {
    id: '2',
    role: 'assistant',
    content: '# 回答',
    sources: [{ id: 'c1', title: '来源', snippet: '片段' }],
    tokensUsed: 10,
    createdAt: '2026-01-01T00:00:01Z',
  },
];

describe('MessageList', () => {
  it('渲染用户与助手消息及来源', () => {
    render(<MessageList messages={msgs} />);
    expect(screen.getByText('问题')).toBeInTheDocument();
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('回答');
    expect(screen.getByText('引用来源')).toBeInTheDocument();
  });
});
