import { useCallback, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { SparkleIcon } from '@/primitives/icons';
import { useConversationStore } from '@/stores/conversationStore';
import { useUiStore } from '@/stores/uiStore';
import type { Message } from '@/api/conversation/types';
import { MarkdownRenderer } from './MarkdownRenderer';
import SourcesCard from './SourcesCard';
import styles from './MessageBubble.module.css';

export default function MessageBubble({
  message,
  streaming,
}: {
  message: Pick<Message, 'id' | 'role' | 'content' | 'sources'>;
  streaming?: boolean;
}) {
  const { t } = useTranslation();
  const isUser = message.role === 'user';
  const [sourcesExpanded, setSourcesExpanded] = useState(false);
  const sourceRefs = useRef<(HTMLDivElement | null)[]>([]);
  const setSelectedMessage = useConversationStore((s) => s.setSelectedMessage);
  const setHighlightSource = useConversationStore((s) => s.setHighlightSource);
  const setDetailsOpen = useUiStore((s) => s.setDetailsOpen);

  const handleSourceRef = useCallback(
    (n: number) => {
      setSelectedMessage(message as Message);
      setHighlightSource(n);
      setDetailsOpen(true);
      setSourcesExpanded(true);
      requestAnimationFrame(() => {
        const el = sourceRefs.current[n - 1];
        if (!el) return;
        el.scrollIntoView({ behavior: 'smooth', block: 'center' });
        el.classList.add('source-highlight');
        setTimeout(() => el.classList.remove('source-highlight'), 2200);
      });
    },
    [message, setSelectedMessage, setHighlightSource, setDetailsOpen],
  );

  if (isUser) {
    return (
      <div className={styles.userRow}>
        <div className={styles.userBubble}>{message.content}</div>
      </div>
    );
  }

  return (
    <div className={styles.assistantRow}>
      <div className={styles.avatar}>
        <SparkleIcon size={15} />
      </div>
      <div className={styles.assistantBody}>
        <div className="md-body">
          <MarkdownRenderer
            content={message.content + (streaming ? t('message.streaming.cursor') : '')}
            sources={message.sources}
            onSourceRef={handleSourceRef}
          />
        </div>
        {message.sources && message.sources.length > 0 && (
          <div className={styles.sources}>
            <SourcesCard
              sources={message.sources}
              expanded={sourcesExpanded}
              onToggle={setSourcesExpanded}
              registerSourceRef={(i, el) => {
                sourceRefs.current[i] = el;
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
}
