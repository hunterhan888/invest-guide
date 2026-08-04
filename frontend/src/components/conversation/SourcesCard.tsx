import { Collapse, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import type { KnowledgeChunkRef } from '@/api/conversation/types';

type Props = {
  sources: KnowledgeChunkRef[] | null;
  expanded?: boolean;
  onToggle?: (expanded: boolean) => void;
  registerSourceRef?: (index: number, el: HTMLDivElement | null) => void;
};

export default function SourcesCard({ sources, expanded, onToggle, registerSourceRef }: Props) {
  const { t } = useTranslation();
  if (!sources || sources.length === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('message.sources.empty')} />;
  }
  return (
    <Collapse
      ghost
      size="small"
      className="sources-card"
      activeKey={expanded ? ['src'] : []}
      onChange={(keys) => onToggle?.(keys.length > 0)}
      expandIconPosition="end"
      items={[
        {
          key: 'src',
          label: (
            <span className="inline-flex items-center gap-1.5">
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                className="text-fg-tertiary"
              >
                <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
                <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
              </svg>
              <span>{t('message.sources.title')}</span>
              <span className="text-fg-tertiary">·</span>
              <span className="text-fg-tertiary truncate max-w-[200px]">
                {sources[0]?.title ?? ''}
              </span>
            </span>
          ),
          forceRender: true,
          children: (
            <div className="flex flex-col gap-0.5">
              {sources.map((s, i) => (
                <div
                  key={s.id}
                  id={`src-${i + 1}`}
                  ref={(el) => registerSourceRef?.(i, el)}
                  className="source-item rounded-lg px-2.5 py-2"
                >
                  <div className="flex gap-2.5">
                    <span className="mt-0.5 inline-flex h-5 w-5 shrink-0 items-center justify-center rounded bg-primary-soft text-xs font-semibold text-primary">
                      {i + 1}
                    </span>
                    <div className="min-w-0">
                      {s.title && (
                        <div className="text-xs font-medium text-fg-secondary mb-0.5 truncate">
                          {s.title}
                        </div>
                      )}
                      <div className="text-sm leading-[1.65] text-fg">{s.snippet}</div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ),
        },
      ]}
    />
  );
}
