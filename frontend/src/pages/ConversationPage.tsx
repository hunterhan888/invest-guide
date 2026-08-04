import { useEffect, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { Button, Spin, Typography } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useConversation } from '@/hooks/useConversation';
import { useConversationStore } from '@/stores/conversationStore';
import MessageList from '@/components/conversation/MessageList';
import { MessageComposer } from '@/components/conversation/MessageComposer';
import type { Message } from '@/api/conversation/types';
import type { Paginated } from '@/api/types';

type MessagesLike = { items: Message[] };
type ComposerMutate = (
  updater?: ((cur?: MessagesLike) => MessagesLike) | MessagesLike,
  opts?: { revalidate?: boolean },
) => Promise<unknown>;

export default function ConversationPage() {
  const { id } = useParams();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { conv, messages, mutateMessages } = useConversation(id ?? null);
  const setActive = useConversationStore((s) => s.setActive);
  const clearActive = useConversationStore((s) => s.clearActive);
  const [streamingId, setStreamingId] = useState<string | null>(null);

  // 首页发起的首个问题会携带 pendingMessageId 跳转；这里消费后清掉 history state，
  // 避免刷新/返回时重复触发。MessageComposer 按 id 重挂载，惰性初始化拿到该值。
  useEffect(() => {
    if (location.state?.pendingMessageId) {
      window.history.replaceState({}, '', location.pathname);
    }
  }, [location]);

  useEffect(() => {
    if (id) setActive(id);
  }, [id, setActive]);

  if (conv.error) return <Typography.Text type="danger">{t('error.generic')}</Typography.Text>;
  if (!conv.data) return <Spin />;

  const composerMutate: ComposerMutate = (updater, opts) => {
    if (typeof updater === 'function') {
      return mutateMessages((cur?: Paginated<Message>) => {
        const next = updater(cur);
        return {
          items: next.items,
          total: cur?.total ?? next.items.length,
          hasMore: cur?.hasMore ?? false,
        };
      }, opts);
    }
    if (updater) {
      return mutateMessages(
        { items: updater.items, total: updater.items.length, hasMore: false },
        opts,
      );
    }
    return mutateMessages();
  };

  return (
    <div className="h-full flex flex-col">
      <header className="flex h-14 shrink-0 items-center justify-between border-b border-border px-6">
        <span className="truncate text-[16px] font-semibold text-fg">{conv.data.title}</span>
        <Button
          type="text"
          icon={<PlusOutlined />}
          onClick={() => {
            clearActive();
            navigate('/');
          }}
        >
          {t('home.newButton')}
        </Button>
      </header>
      <MessageList messages={messages.data?.items ?? []} streamingId={streamingId ?? undefined} />
      <MessageComposer
        key={id}
        conversationId={id!}
        messages={{ items: messages.data?.items ?? [] }}
        mutateMessages={composerMutate}
        onStreamingChange={setStreamingId}
        initialStreamMessageId={
          (location.state as { pendingMessageId?: string } | null)?.pendingMessageId ?? null
        }
      />
    </div>
  );
}
