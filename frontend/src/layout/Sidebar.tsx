import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Modal } from '@/primitives/Modal';
import { Tooltip } from '@/primitives/Tooltip';
import { PlusIcon, MoreIcon, DeleteIcon } from '@/primitives/icons';
import { useConversations } from '@/hooks/useConversations';
import { useConversationStore } from '@/stores/conversationStore';
import { deleteConversation } from '@/api/conversation/conversation';
import type { Conversation } from '@/api/conversation/types';
import { CompassLogo } from '@/theme/logo';
import styles from './Sidebar.module.css';

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

export default function Sidebar({ collapsed }: { collapsed?: boolean }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams();
  const { data, mutate } = useConversations();
  const clearActive = useConversationStore((s) => s.clearActive);
  const [pendingDelete, setPendingDelete] = useState<Conversation | null>(null);
  const [deleting, setDeleting] = useState(false);

  const groups = useMemo(() => groupConversations(data?.items ?? []), [data]);

  function goHome() {
    clearActive();
    navigate('/');
  }

  async function remove(convId: string) {
    setDeleting(true);
    try {
      await deleteConversation(convId);
      await mutate();
      if (id === convId) goHome();
      setPendingDelete(null);
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className={styles.root} data-collapsed={collapsed || undefined}>
      <div className={styles.logoRow}>
        <button type="button" className={styles.brand} onClick={goHome}>
          <CompassLogo size={24} />
          {!collapsed && <span className={styles.brandText}>{t('sidebar.title')}</span>}
        </button>
      </div>

      <Button
        variant="outline"
        block
        className={styles.newSession}
        icon={<PlusIcon size={14} />}
        onClick={goHome}
      >
        {!collapsed && t('sidebar.newConversation')}
      </Button>

      <div className={styles.regionArea}>
        {groups.map((group) => (
          <div key={group.key} className={styles.group}>
            <div className={styles.groupLabel}>{t(`sidebar.group.${group.key}`)}</div>
            {group.items.map((conv) => {
              const active = conv.id === id;
              return (
                <div key={conv.id} className={styles.rowWrap}>
                  <button
                    type="button"
                    className={`${styles.item} ${active ? styles.active : ''}`.trim()}
                    aria-current={active ? 'page' : undefined}
                    onClick={() => navigate(`/conversations/${conv.id}`)}
                  >
                    <span className={styles.itemTitle}>{conv.title}</span>
                  </button>
                  <Tooltip content={t('common.delete')}>
                    <button
                      type="button"
                      className={styles.moreBtn}
                      aria-label={t('common.delete')}
                      onClick={(e) => {
                        e.stopPropagation();
                        setPendingDelete(conv);
                      }}
                    >
                      <MoreIcon size={14} />
                    </button>
                  </Tooltip>
                </div>
              );
            })}
          </div>
        ))}
      </div>

      <Modal
        open={pendingDelete != null}
        title={t('conversation.list.deleteConfirm')}
        onClose={() => setPendingDelete(null)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setPendingDelete(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="primary"
              icon={<DeleteIcon size={14} />}
              loading={deleting}
              onClick={() => pendingDelete && void remove(pendingDelete.id)}
            >
              {t('common.confirm')}
            </Button>
          </>
        }
      />
    </div>
  );
}
