import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DetailsPanel } from './DetailsPanel';
import { useConversationStore } from '@/stores/conversationStore';

const msg = {
  id: 'm1',
  role: 'assistant' as const,
  content: '回答',
  sources: [{ id: 'c1', title: '来源一', snippet: '片段一' }],
  tokensUsed: 123,
  createdAt: '2026-01-01T00:00:00Z',
};

describe('DetailsPanel', () => {
  beforeEach(() => {
    useConversationStore.getState().setSelectedMessage(null);
    useConversationStore.getState().setHighlightSource(null);
  });

  it('无选中消息时显示空态', () => {
    render(<DetailsPanel />);
    expect(screen.getByText(/引用详情/)).toBeInTheDocument();
  });

  it('展示选中消息的来源与元信息', () => {
    useConversationStore.getState().setSelectedMessage(msg);
    render(<DetailsPanel />);
    expect(screen.getByText('来源一')).toBeInTheDocument();
    expect(screen.getByText(/123/)).toBeInTheDocument();
  });
});
