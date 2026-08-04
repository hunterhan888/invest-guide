import { useState } from 'react';
import { Popover } from 'antd';
import type { KnowledgeChunkRef } from '@/api/conversation/types';

type Props = {
  n: number;
  source: KnowledgeChunkRef | undefined;
  onSourceRef?: (n: number) => void;
};

/**
 * NotebookLM 风格的内联引用芯片：
 * - 小圆角方形，内嵌数字编号
 * - 点击后弹出 Popover 显示来源片段
 * - 同时触发外部的 onSourceRef（滚动到下方 SourcesCard 对应条目）
 */
export default function SourceChip({ n, source, onSourceRef }: Props) {
  const [open, setOpen] = useState(false);

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setOpen((prev) => !prev);
    onSourceRef?.(n);
  };

  const chip = (
    <span
      className="source-chip"
      onClick={handleClick}
      role="button"
      tabIndex={0}
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

  if (!source?.snippet) return chip;

  return (
    <Popover
      open={open}
      onOpenChange={setOpen}
      trigger="click"
      placement="top"
      overlayStyle={{ maxWidth: 360 }}
      title={
        <div className="flex items-center gap-2">
          <span className="inline-flex h-5 w-5 items-center justify-center rounded bg-primary-soft text-xs font-semibold text-primary">
            {n}
          </span>
          <span className="text-xs font-medium text-fg-secondary truncate max-w-[260px]">
            {source.title ?? `来源 ${n}`}
          </span>
        </div>
      }
      content={
        <div className="max-h-40 overflow-y-auto text-sm leading-relaxed text-fg">
          {source.snippet}
        </div>
      }
    >
      {chip}
    </Popover>
  );
}
