import { request } from '../client';
import type { Paginated } from '../types';
import type {
  Conversation,
  CreateConversationRequest,
  Message,
  SendMessageRequest,
  SendMessageResponse,
} from './types';

export function listConversations(page = 1, pageSize = 20): Promise<Paginated<Conversation>> {
  return request<Paginated<Conversation>>(
    'GET',
    `/conversations?page=${page}&pageSize=${pageSize}`,
  );
}

export function getConversation(id: string): Promise<Conversation> {
  return request<Conversation>('GET', `/conversations/${id}`);
}

export function createConversation(req: CreateConversationRequest): Promise<Conversation> {
  return request<Conversation>('POST', '/conversations', req);
}

export function deleteConversation(id: string): Promise<void> {
  return request<void>('DELETE', `/conversations/${id}`);
}

export function listMessages(convId: string): Promise<Paginated<Message>> {
  return request<Paginated<Message>>('GET', `/conversations/${convId}/messages?page=1&pageSize=50`);
}

export function sendMessage(convId: string, req: SendMessageRequest): Promise<SendMessageResponse> {
  return request<SendMessageResponse>('POST', `/conversations/${convId}/messages`, req);
}
