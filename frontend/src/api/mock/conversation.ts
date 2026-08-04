import { registerMock, type MockRequest } from '../client';
import { jsonOk, jsonFail } from './tools';
import type { Conversation, Message } from '../conversation/types';

type Store = { conversations: Map<string, Conversation>; messages: Map<string, Message[]> };
const store: Store = { conversations: new Map(), messages: new Map() };

function newId() {
  return Math.random().toString(36).slice(2);
}

function ts() {
  return new Date().toISOString();
}

export function __resetConversationData() {
  store.conversations.clear();
  store.messages.clear();
}

function parseQuery(path: string) {
  const u = new URL('http://x' + path);
  return {
    page: Number(u.searchParams.get('page') ?? '1'),
    pageSize: Number(u.searchParams.get('pageSize') ?? '20'),
  };
}

export function installConversationMocks() {
  registerMock({
    match: (r) => r.method === 'GET' && r.path.startsWith('/conversations?'),
    handle: async (r: MockRequest) => {
      const { page, pageSize } = parseQuery(r.path);
      const all = [...store.conversations.values()].sort((a, b) =>
        b.updatedAt.localeCompare(a.updatedAt),
      );
      const start = (page - 1) * pageSize;
      const items = all.slice(start, start + pageSize);
      return jsonOk({ items, total: all.length, hasMore: start + pageSize < all.length });
    },
  });

  registerMock({
    match: (r) => r.method === 'POST' && r.path === '/conversations',
    handle: async (r: MockRequest) => {
      const body = (r.body ?? {}) as { title?: string; country?: string };
      const conv: Conversation = {
        id: newId(),
        title: body.title ?? '新会话',
        country: body.country ?? null,
        createdAt: ts(),
        updatedAt: ts(),
      };
      store.conversations.set(conv.id, conv);
      store.messages.set(conv.id, []);
      return jsonOk(conv, 201);
    },
  });

  registerMock({
    match: (r) => r.method === 'GET' && /^\/conversations\/[^/]+\/?$/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      const conv = store.conversations.get(id);
      if (!conv) return jsonFail('NOT_FOUND', 'not found', 404);
      return jsonOk(conv);
    },
  });

  registerMock({
    match: (r) => r.method === 'DELETE' && /^\/conversations\/[^/]+\/?$/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      if (!store.conversations.has(id)) return jsonFail('NOT_FOUND', 'not found', 404);
      store.conversations.delete(id);
      store.messages.delete(id);
      return jsonOk(null, 204);
    },
  });

  registerMock({
    match: (r) => r.method === 'GET' && /^\/conversations\/[^/]+\/messages(\?|$)/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      if (!store.conversations.has(id)) return jsonFail('NOT_FOUND', 'not found', 404);
      const msgs = store.messages.get(id) ?? [];
      return jsonOk({ items: msgs, total: msgs.length, hasMore: false });
    },
  });

  registerMock({
    match: (r) => r.method === 'POST' && /\/conversations\/[^/]+\/messages\/?$/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      const conv = store.conversations.get(id);
      if (!conv) return jsonFail('NOT_FOUND', 'not found', 404);
      const body = r.body as { content: string };

      const userMsg: Message = {
        id: newId(),
        role: 'user',
        content: body.content,
        sources: null,
        tokensUsed: null,
        createdAt: ts(),
      };
      const assistantMsg: Message = {
        id: newId(),
        role: 'assistant',
        content: '',
        sources: null,
        tokensUsed: null,
        createdAt: ts(),
      };
      store.messages.get(id)?.push(userMsg, assistantMsg);

      if (conv.title === '新会话') {
        conv.title = body.content.length > 20 ? body.content.slice(0, 20) + '...' : body.content;
      }
      conv.updatedAt = ts();

      return jsonOk({ messageId: assistantMsg.id }, 201);
    },
  });

  registerMock({
    match: (r) =>
      r.method === 'GET' && /\/conversations\/[^/]+\/messages\/[^/]+\/stream$/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      const msgs = store.messages.get(id) ?? [];
      const lastUser = [...msgs].reverse().find((m) => m.role === 'user');
      const { answerStream, errorStream } = await import('./sse');
      if (r.path.includes('force-error')) return errorStream();
      return answerStream(lastUser?.content ?? '');
    },
  });
}
