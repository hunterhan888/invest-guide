import { sseStream, type SSEFrame } from './tools';
import type { MockResponse } from '../client';

export function answerStream(questionContent: string): MockResponse {
  const body = `针对「${questionContent}」，这是模拟的流式回答。支持 **Markdown**：\n\n- 列表项 A\n- 列表项 B\n\n以及代码块。`;
  const chunks = body.split('');
  const frames: SSEFrame[] = [
    { event: 'heartbeat', data: {} },
    {
      event: 'sources',
      data: {
        chunks: [{ id: 'chunk-1', title: '对外投资合作国别指南 - 越南', snippet: '示例来源片段…' }],
      },
    },
    ...chunks.map((c) => ({ event: 'message', data: { delta: c } })),
    { event: 'done', data: { messageId: 'mock_done', tokensUsed: chunks.length } },
  ];
  return sseStream(frames);
}

export function errorStream(code = 'LLM_TIMEOUT'): MockResponse {
  return sseStream([{ event: 'error', data: { code, message: 'stream failed' } }]);
}
