import type { KnowledgeChunkRef } from '@/api/conversation/types';

type Props = {
  n: number;
  source: KnowledgeChunkRef | undefined;
  onSourceRef?: (n: number) => void;
};

export default function SourceChip({ n, source, onSourceRef }: Props) {
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onSourceRef?.(n);
  };

  return (
    <span
      className="source-chip"
      onClick={handleClick}
      role="button"
      tabIndex={0}
      title={source?.title ?? `来源 ${n}`}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          handleClick(e as unknown as React.MouseEvent);
        }
      }}
    >
      {n}
    </span>
  );
}
