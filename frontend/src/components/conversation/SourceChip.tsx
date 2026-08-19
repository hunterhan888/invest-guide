import { useTranslation } from 'react-i18next';
import type { KnowledgeChunkRef } from '@/api/conversation/types';

type Props = {
  n: number;
  source: KnowledgeChunkRef | undefined;
  onSourceRef?: (n: number) => void;
};

export default function SourceChip({ n, source, onSourceRef }: Props) {
  const { t } = useTranslation();

  const activate = (e: React.SyntheticEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onSourceRef?.(n);
  };

  return (
    <span
      className="source-chip"
      onClick={activate}
      role="button"
      tabIndex={0}
      aria-label={source?.title ?? t('message.sources.fallback', { n })}
      title={source?.title ?? t('message.sources.fallback', { n })}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          activate(e);
        }
      }}
    >
      {n}
    </span>
  );
}
