import { describe, it, expect, beforeEach } from 'vitest';
import { __resetMocks } from '../client';
import { installConversationMocks, __resetConversationData } from '../mock/conversation';
import {
  listConversations,
  createConversation,
  getConversation,
  deleteConversation,
  listMessages,
  sendMessage,
} from './conversation';

describe('conversation api', () => {
  beforeEach(() => {
    __resetMocks();
    installConversationMocks();
    __resetConversationData();
  });

  it('createConversation 后能 list 出来', async () => {
    const created = await createConversation({ title: '首问' });
    const list = await listConversations(1);
    expect(list.items).toHaveLength(1);
    expect(list.items[0]?.id).toBe(created.id);
  });

  it('listConversations 按 updatedAt 倒序', async () => {
    const a = await createConversation({ title: 'A' });
    const b = await createConversation({ title: 'B' });
    // b 更新较晚，应排在前面
    await sendMessage(a.id, { content: '在 A 里说话' });
    const list = await listConversations(1);
    expect(list.items[0]?.id).toBe(a.id);
    expect(list.items[1]?.id).toBe(b.id);
  });

  it('sendMessage 返回 messageId 且消息历史含 user 消息', async () => {
    const conv = await createConversation({});
    const { messageId } = await sendMessage(conv.id, { content: '你好' });
    expect(messageId).toBeTypeOf('string');
    const msgs = await listMessages(conv.id);
    expect(msgs.items.some((m) => m.role === 'user')).toBe(true);
  });

  it('listConversations 分页 hasMore 边界正确', async () => {
    for (let i = 0; i < 25; i++) await createConversation({});
    const p1 = await listConversations(1, 20);
    expect(p1.items).toHaveLength(20);
    expect(p1.hasMore).toBe(true);
    const p2 = await listConversations(2, 20);
    expect(p2.items).toHaveLength(5);
    expect(p2.hasMore).toBe(false);
  });

  it('deleteConversation 后 getConversation 抛 404', async () => {
    const conv = await createConversation({});
    await deleteConversation(conv.id);
    await expect(getConversation(conv.id)).rejects.toMatchObject({ status: 404 });
  });
});
