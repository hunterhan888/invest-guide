export type MessageRole = 'user' | 'assistant';

export type KnowledgeChunkRef = {
  id: string;
  title: string;
  snippet: string;
};

export type Conversation = {
  id: string;
  title: string;
  country: string | null;
  createdAt: string;
  updatedAt: string;
};

export type Message = {
  id: string;
  role: MessageRole;
  content: string;
  sources: KnowledgeChunkRef[] | null;
  tokensUsed: number | null;
  createdAt: string;
};

export type CreateConversationRequest = { title?: string; country?: string };
export type SendMessageRequest = { content: string };
export type SendMessageResponse = { messageId: string };
