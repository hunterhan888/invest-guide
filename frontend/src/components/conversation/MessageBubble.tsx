import { useCallback, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RobotOutlined } from '@ant-design/icons';
import type { Message } from '@/api/conversation/types';
import { MarkdownRenderer } from './MarkdownRenderer';
import SourcesCard from './SourcesCard';

export default function MessageBubble({
  message,
  streaming,
}: {
  message: Pick<Message, 'role' | 'content' | 'sources'>;
  streaming?: boolean;
}) {
  const { t } = useTranslation();
  const isUser = message.role === 'user';
  const [sourcesExpanded, setSourcesExpanded] = useState(false);
  const sourceRefs = useRef<(HTMLDivElement | null)[]>([]);

  // 点击回答里的「片段[N]」→ 展开来源卡并滚动高亮第 N 条来源
  const handleSourceRef = useCallback((n: number) => {
    setSourcesExpanded(true);
    requestAnimationFrame(() => {
      const el = sourceRefs.current[n - 1];
      if (!el) return;
      el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      el.classList.add('source-highlight');
      setTimeout(() => el.classList.remove('source-highlight'), 2200);
    });
  }, []);

  if (isUser) {
    return (
      <div className="flex justify-end">
        <div className="max-w-[75%] rounded-[16px] bg-primary-soft px-4 py-2.5 whitespace-pre-wrap break-words text-[15px] leading-[1.6] text-fg">
          {message.content}
        </div>
      </div>
    );
  }

  return (
    <div className="flex justify-start">
      <div
        className="mr-3 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg"
        style={{ background: 'linear-gradient(135deg, #4096ff, #1677ff)' }}
      >
        <RobotOutlined style={{ fontSize: 16, color: '#fff' }} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="md-body">
          <MarkdownRenderer
            content={message.content + (streaming ? t('message.streaming.cursor') : '')}
            sources={message.sources}
            onSourceRef={handleSourceRef}
          />
        </div>
        {message.sources && message.sources.length > 0 && (
          <div className="mt-3">
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
