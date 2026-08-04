import { useState } from 'react';
import { App as AntdApp, Button, Input, Typography } from 'antd';
import { ArrowUpOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { createConversation, sendMessage } from '@/api/conversation/conversation';
import { useConversationStore } from '@/stores/conversationStore';

export default function HomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = AntdApp.useApp();
  const setActive = useConversationStore((s) => s.setActive);
  const [value, setValue] = useState('');
  const [loading, setLoading] = useState(false);

  async function ask() {
    const content = value.trim();
    if (!content || loading) return;
    setLoading(true);
    try {
      const conv = await createConversation({});
      setActive(conv.id);
      const { messageId } = await sendMessage(conv.id, { content });
      navigate(`/conversations/${conv.id}`, { state: { pendingMessageId: messageId } });
    } catch {
      message.error(t('error.generic'));
      setLoading(false);
    }
  }

  return (
    <div className="flex h-full flex-col items-center justify-center px-6">
      <div className="mb-8 flex flex-col items-center">
        <div className="mb-4 text-[48px] leading-none">🤖</div>
        <Typography.Title level={2} className="!mb-0 !font-bold !text-fg">
          {t('home.welcome')}
        </Typography.Title>
      </div>
      <div className="relative w-full max-w-[800px] rounded-[16px] border border-[#d9d9d9] bg-white">
        <Input.TextArea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={t('composer.placeholder')}
          autoSize={{ minRows: 3, maxRows: 8 }}
          className="!border-none !bg-transparent !px-5 !py-4 !text-[16px] !shadow-none"
          onPressEnter={(e) => {
            if (!e.shiftKey) {
              e.preventDefault();
              void ask();
            }
          }}
        />
        <div className="flex justify-end p-3 pt-0">
          <Button
            type="primary"
            shape="circle"
            icon={<ArrowUpOutlined />}
            loading={loading}
            disabled={!value.trim()}
            onClick={() => void ask()}
          />
        </div>
      </div>
    </div>
  );
}
