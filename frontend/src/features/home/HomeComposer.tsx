import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Textarea } from '@/primitives/Textarea';
import { SendIcon, StopIcon } from '@/primitives/icons';
import { createConversation, sendMessage } from '@/api/conversation/conversation';
import { useConversationStore } from '@/stores/conversationStore';
import { useToast } from '@/primitives/ToastProvider';
import styles from './HomeComposer.module.css';

const MAX_LEN = 2000;

export function HomeComposer() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const setActive = useConversationStore((s) => s.setActive);
  const [value, setValue] = useState('');
  const [loading, setLoading] = useState(false);

  async function ask() {
    const content = value.trim();
    if (!content || loading) return;
    if (content.length > MAX_LEN) {
      toast.error(t('composer.tooLong'));
      return;
    }
    setLoading(true);
    try {
      const conv = await createConversation({});
      setActive(conv.id);
      const { messageId } = await sendMessage(conv.id, { content });
      navigate(`/conversations/${conv.id}`, { state: { pendingMessageId: messageId } });
    } catch {
      toast.error(t('error.generic'));
      setLoading(false);
    }
  }

  return (
    <div className={styles.card}>
      <Textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder={t('composer.placeholder')}
        autoSize={{ minRows: 3, maxRows: 8 }}
        disabled={loading}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            void ask();
          }
        }}
      />
      <div className={styles.actions}>
        {loading ? (
          <Button variant="primary" size="sm" icon={<StopIcon size={14} />} onClick={() => setLoading(false)}>
            {t('common.stop')}
          </Button>
        ) : (
          <Button
            variant="primary"
            size="sm"
            icon={<SendIcon size={14} />}
            disabled={!value.trim()}
            onClick={() => void ask()}
          >
            {t('common.send')}
          </Button>
        )}
      </div>
    </div>
  );
}
