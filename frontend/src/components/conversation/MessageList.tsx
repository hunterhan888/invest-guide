import { useEffect, useRef } from 'react';
import type { Message } from '@/api/conversation/types';
import MessageBubble from './MessageBubble';

export default function MessageList({
  messages,
  streamingId,
}: {
  messages: Message[];
  streamingId?: string;
}) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const atBottomRef = useRef(true);

  function onScroll() {
    const el = containerRef.current;
    if (!el) return;
    atBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
  }

  useEffect(() => {
    if (atBottomRef.current) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  return (
    <div ref={containerRef} onScroll={onScroll} className="flex-1 overflow-auto px-4 py-6">
      <div className="mx-auto w-full max-w-[680px] space-y-6">
        {messages.map((m) => (
          <MessageBubble key={m.id} message={m} streaming={m.id === streamingId} />
        ))}
      </div>
      <div className="mx-auto w-full max-w-[680px]" ref={bottomRef} />
    </div>
  );
}
