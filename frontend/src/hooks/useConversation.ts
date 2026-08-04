import useSWR from 'swr';
import { getConversation, listMessages } from '@/api/conversation/conversation';
import { useAuthStore } from '@/stores/authStore';

export function useConversation(id: string | null) {
  const token = useAuthStore((s) => s.token);
  const conv = useSWR(id && token ? ['conversation', id, token] : null, () => getConversation(id!));
  const messages = useSWR(id && token ? ['messages', id, token] : null, () => listMessages(id!));
  return { conv, messages, mutateMessages: messages.mutate };
}
