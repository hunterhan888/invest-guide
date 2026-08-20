import { useTranslation } from 'react-i18next';
import { DisclosureRow } from '@/primitives/DisclosureRow';
import { DocumentIcon } from '@/primitives/icons';
import type { KnowledgeChunkRef } from '@/api/conversation/types';
import styles from './SourcesCard.module.css';

type Props = {
  sources: KnowledgeChunkRef[] | null;
  expanded?: boolean;
  onToggle?: (expanded: boolean) => void;
  registerSourceRef?: (index: number, el: HTMLDivElement | null) => void;
};

export default function SourcesCard({ sources, expanded, onToggle, registerSourceRef }: Props) {
  const { t } = useTranslation();
  if (!sources || sources.length === 0) {
    return <div className={styles.empty}>{t('message.sources.empty')}</div>;
  }
  return (
    <DisclosureRow
      title={
        <span className={styles.label}>
          <DocumentIcon size={13} />
          <span>{t('message.sources.title')}</span>
          {sources[0]?.title && <span className={styles.preview}> · {sources[0].title}</span>}
        </span>
      }
      expanded={expanded}
      onToggle={onToggle}
    >
      <div className={styles.list}>
        {sources.map((s, i) => (
          <div
            key={s.id ?? `source-${i}`}
            id={`src-${i + 1}`}
            ref={(el) => registerSourceRef?.(i, el)}
            className={styles.item}
          >
            <div className={styles.itemHead}>
              <span className={styles.index}>{i + 1}</span>
              {s.title && <span className={styles.itemTitle}>{s.title}</span>}
            </div>
            <div className={styles.snippet}>{s.snippet}</div>
          </div>
        ))}
      </div>
    </DisclosureRow>
  );
}
