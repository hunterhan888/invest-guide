import { useMemo } from 'react';
import { Button, Popconfirm, Tooltip } from 'antd';
import { PlusOutlined, MoreOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useConversations } from '@/hooks/useConversations';
import { useConversationStore } from '@/stores/conversationStore';
import { deleteConversation } from '@/api/conversation/conversation';
import type { Conversation } from '@/api/conversation/types';

type GroupKey = 'today' | 'yesterday' | 'earlier';

function groupKeyOf(iso: string, now: Date): GroupKey {
  const d = new Date(iso);
  const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const startOfYesterday = startOfDay - 24 * 60 * 60 * 1000;
  const t = d.getTime();
  if (t >= startOfDay) return 'today';
  if (t >= startOfYesterday) return 'yesterday';
  return 'earlier';
}

function groupConversations(
  items: Conversation[],
  now = new Date(),
): Array<{ key: GroupKey; items: Conversation[] }> {
  const groups: Record<GroupKey, Conversation[]> = { today: [], yesterday: [], earlier: [] };
  for (const c of items) {
    groups[groupKeyOf(c.updatedAt, now)].push(c);
  }
  return (Object.keys(groups) as GroupKey[]).map((key) => ({ key, items: groups[key] }));
}

export default function Sidebar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams();
  const { data, mutate } = useConversations();
  const clearActive = useConversationStore((s) => s.clearActive);

  const groups = useMemo(() => groupConversations(data?.items ?? []), [data]);

  function goHome() {
    clearActive();
    navigate('/');
  }

  async function remove(convId: string) {
    await deleteConversation(convId);
    await mutate();
    if (id === convId) goHome();
  }

  return (
    <div className="flex h-full flex-col">
      <div className="p-2">
        <Button
          className="!h-11 !border-none !bg-[#e6f7ff] !text-[20px]"
          block
          icon={<PlusOutlined />}
          onClick={goHome}
        >
          {t('sidebar.newConversation')}
        </Button>
      </div>
      <div className="flex-1 overflow-y-auto px-2 pb-2">
        {groups.map((group) => (
          <div key={group.key} className="mb-2">
            <div className="px-2 py-1 text-[16px] font-medium text-fg-tertiary">
              {t(`sidebar.group.${group.key}`)}
            </div>
            {group.items.map((conv) => {
              const active = conv.id === id;
              return (
                <div
                  key={conv.id}
                  className={`group flex cursor-pointer items-center rounded-lg ${active ? 'bg-[#e6f7ff]' : 'hover:bg-[#f5f5f5]'}`}
                  onClick={() => {
                    navigate(`/conversations/${conv.id}`);
                  }}
                >
                  <span className="flex-1 truncate py-2 pl-2 text-[18px] text-fg">
                    {conv.title}
                  </span>
                  <Popconfirm
                    title={t('conversation.list.deleteConfirm')}
                    okText={t('common.confirm')}
                    cancelText={t('common.cancel')}
                    onConfirm={() => void remove(conv.id)}
                  >
                    <Tooltip title={t('common.delete')}>
                      <Button
                        type="text"
                        size="small"
                        className="mr-1 hidden group-hover:inline-flex"
                        icon={<MoreOutlined />}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </Tooltip>
                  </Popconfirm>
                </div>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
}
