import { useRef, useState } from 'react';
import { Button, Input } from 'antd';
import { SendOutlined, StopOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { sendMessage as apiSendMessage } from '@/api/conversation/conversation';
import { useSSEStream, type SSEEvent } from '@/hooks/useSSEStream';
import type { Message } from '@/api/conversation/types';

type MessagesLike = { items: Message[] };
type MutateFn = (
  updater?: ((cur?: MessagesLike) => MessagesLike) | MessagesLike,
  opts?: { revalidate?: boolean },
) => Promise<unknown>;

type Props = {
  conversationId: string;
  messages: MessagesLike;
  mutateMessages: MutateFn;
  onStreamingChange?: (id: string | null) => void;
  initialStreamMessageId?: string | null;
};

const MAX_LEN = 2000;

export function MessageComposer({
  conversationId,
  messages,
  mutateMessages,
  onStreamingChange,
  initialStreamMessageId,
}: Props) {
  const { t } = useTranslation();
  const [value, setValue] = useState('');
  const [activeMessageId, setActiveMessageId] = useState<string | null>(
    initialStreamMessageId ?? null,
  );
  const [errorReason, setErrorReason] = useState<string | null>(null);
  const lastUserContent = useRef('');

  function setActive(id: string | null) {
    setActiveMessageId(id);
    onStreamingChange?.(id);
  }

  function onEvent(e: SSEEvent) {
    if (e.type === 'heartbeat') return;
    if (e.type === 'sources') {
      void mutateMessages(
        (cur) => {
          const items = cur?.items ?? [];
          const idx = items.findIndex((m) => m.id === activeMessageId);
          if (idx === -1) return cur ?? { items };
          const copy = [...items];
          copy[idx] = { ...copy[idx]!, sources: e.chunks };
          return { items: copy };
        },
        { revalidate: false },
      );
      return;
    }
    if (e.type === 'message') {
      void mutateMessages(
        (cur) => {
          const items = cur?.items ?? [];
          const idx = items.findIndex((m) => m.id === activeMessageId);
          if (idx === -1) return cur ?? { items };
          const copy = [...items];
          copy[idx] = { ...copy[idx]!, content: copy[idx]!.content + e.delta };
          return { items: copy };
        },
        { revalidate: false },
      );
    } else if (e.type === 'done') {
      setActive(null);
    } else if (e.type === 'error') {
      setErrorReason(e.message);
      setActive(null);
    }
  }

  // 首页带入的待流式消息：等它出现在消息列表中再开流，避免事件先于列表加载而丢失
  const streamReady = !!activeMessageId && messages.items.some((m) => m.id === activeMessageId);
  const { state, stop } = useSSEStream({
    convId: conversationId,
    messageId: activeMessageId ?? '',
    enabled: streamReady,
    onEvent,
  });

  async function sendContent(content: string) {
    if (!content) return;
    if (content.length > MAX_LEN) {
      setErrorReason(t('composer.tooLong'));
      return;
    }
    setValue('');
    setErrorReason(null);
    lastUserContent.current = content;

    const tempUserId = 'pending_user_' + Date.now();
    const tempAsstId = 'pending_asst_' + Date.now();
    await mutateMessages(
      (cur) => ({
        items: [
          ...(cur?.items ?? messages.items),
          {
            id: tempUserId,
            role: 'user',
            content,
            sources: null,
            tokensUsed: null,
            createdAt: new Date().toISOString(),
          },
          {
            id: tempAsstId,
            role: 'assistant',
            content: '',
            sources: null,
            tokensUsed: null,
            createdAt: new Date().toISOString(),
          },
        ],
      }),
      { revalidate: false },
    );

    try {
      const { messageId } = await apiSendMessage(conversationId, { content });
      await mutateMessages(
        (cur) => {
          const items = (cur?.items ?? []).map((m) =>
            m.id === tempAsstId ? { ...m, id: messageId } : m,
          );
          return { items };
        },
        { revalidate: false },
      );
      setActive(messageId);
    } catch {
      // 回滚乐观消息，避免失败气泡残留
      await mutateMessages(
        (cur) => ({
          items: (cur?.items ?? []).filter((m) => m.id !== tempUserId && m.id !== tempAsstId),
        }),
        { revalidate: false },
      );
      setErrorReason(t('composer.error.reason'));
      setActive(null);
    }
  }

  function send() {
    const content = value.trim();
    if (!content) return;
    if (content.length > MAX_LEN) return;
    void sendContent(content);
  }

  function resend() {
    void sendContent(lastUserContent.current);
  }

  const streaming = state === 'streaming';

  return (
    <div className="px-4 pb-4 pt-2">
      {errorReason && (
        <div className="mx-auto mb-2 w-full max-w-[680px] flex items-center gap-2 text-fg-secondary text-sm">
          <span>
            {t('composer.error.reason')}: {errorReason}
          </span>
          <Button size="small" onClick={() => void resend()}>
            {t('composer.error.retry')}
          </Button>
        </div>
      )}
      <div className="composer-card mx-auto w-full max-w-[680px] px-4 py-3 relative">
        <Input.TextArea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={t('composer.placeholder')}
          autoSize={{ minRows: 1, maxRows: 6 }}
          disabled={streaming}
          className="!bg-transparent !border-none !shadow-none !rounded-none"
          onPressEnter={(e) => {
            if (!e.shiftKey) {
              e.preventDefault();
              send();
            }
          }}
        />
        <div className="flex justify-end mt-2">
          {streaming ? (
            <Button danger icon={<StopOutlined />} onClick={stop} className="!rounded-full">
              {t('composer.stop')}
            </Button>
          ) : (
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={() => void send()}
              disabled={!value.trim()}
              className="!rounded-full"
            >
              {t('composer.send')}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
