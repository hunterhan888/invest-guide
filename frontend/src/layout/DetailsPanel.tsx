import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useConversationStore } from '@/stores/conversationStore';
import { Pill } from '@/primitives/Pill';
import { DocumentIcon, ClockIcon } from '@/primitives/icons';
import styles from './DetailsPanel.module.css';

export function DetailsPanel() {
  const { t } = useTranslation();
  const message = useConversationStore((s) => s.selectedMessage);
  const highlightSource = useConversationStore((s) => s.highlightSource);
  const refs = useRef<(HTMLDivElement | null)[]>([]);

  useEffect(() => {
    if (highlightSource == null) return;
    const el = refs.current[highlightSource - 1];
    if (!el) return;
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    el.classList.add(styles.highlight!);
    const timer = setTimeout(() => el.classList.remove(styles.highlight!), 2200);
    return () => {
      clearTimeout(timer);
      el.classList.remove(styles.highlight!);
    };
  }, [highlightSource]);

  if (!message || !message.sources || message.sources.length === 0) {
    return <div className={styles.empty}>{t('details.empty')}</div>;
  }

  const count = message.sources.length;

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <div className={styles.title}>{t('details.title')}</div>
        <div className={styles.meta}>
          {message.tokensUsed != null && (
            <Pill>{`${t('details.tokens')}: ${message.tokensUsed}`}</Pill>
          )}
          <Pill>{`${count} ${t('message.sources.title')}`}</Pill>
        </div>
      </div>
      <div className={styles.sourceList}>
        {message.sources.map((s, i) => (
          <div
            key={s.id}
            ref={(el) => {
              refs.current[i] = el;
            }}
            className={styles.sourceItem}
          >
            <div className={styles.sourceHead}>
              <span className={styles.sourceIndex}>
                <DocumentIcon size={12} />
                {i + 1}
              </span>
              {s.title && <span className={styles.sourceTitle}>{s.title}</span>}
            </div>
            <div className={styles.sourceSnippet}>{s.snippet}</div>
          </div>
        ))}
      </div>
      <div className={styles.foot}>
        <ClockIcon size={12} />
        <span>{new Date(message.createdAt).toLocaleString()}</span>
      </div>
    </div>
  );
}
