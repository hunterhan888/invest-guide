import { memo, useMemo, type ReactNode } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';
import { linkSourceRefs } from '@/utils/sourceRefs';
import type { KnowledgeChunkRef } from '@/api/conversation/types';
import SourceChip from './SourceChip';

type Props = {
  content: string;
  sources?: KnowledgeChunkRef[] | null;
  onSourceRef?: (n: number) => void;
};

export const MarkdownRenderer = memo(function MarkdownRenderer({
  content,
  sources,
  onSourceRef,
}: Props) {
  const components = useMemo(() => {
    return {
      a: ({ href, children }: { href?: string; children?: ReactNode }) => {
        const m = /^#src-(\d+)$/.exec(href ?? '');
        if (!m) {
          return (
            <a href={href} target="_blank" rel="noopener noreferrer">
              {children}
            </a>
          );
        }
        const n = Number(m[1]);
        const source = sources?.[n - 1];
        return <SourceChip n={n} source={source} onSourceRef={onSourceRef} />;
      },
    };
  }, [sources, onSourceRef]);

  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeHighlight]}
      skipHtml
      components={components}
    >
      {linkSourceRefs(content)}
    </ReactMarkdown>
  );
});
