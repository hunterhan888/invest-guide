import useSWR from 'swr';
import { listConversations } from '@/api/conversation/conversation';
import { useAuthStore } from '@/stores/authStore';

export function useConversations() {
  const token = useAuthStore((s) => s.token);
  return useSWR(token ? ['conversations', token] : null, () => listConversations(1, 50));
}
