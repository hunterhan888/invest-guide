import { create } from 'zustand';
import type { Message } from '@/api/conversation/types';

type ConversationState = {
  activeId: string | null;
  selectedMessage: Message | null;
  highlightSource: number | null;
  setActive: (id: string | null) => void;
  clearActive: () => void;
  setSelectedMessage: (m: Message | null) => void;
  setHighlightSource: (n: number | null) => void;
};

export const useConversationStore = create<ConversationState>((set) => ({
  activeId: null,
  selectedMessage: null,
  highlightSource: null,
  setActive: (id) => set({ activeId: id, selectedMessage: null, highlightSource: null }),
  clearActive: () => set({ activeId: null, selectedMessage: null, highlightSource: null }),
  setSelectedMessage: (m) => set({ selectedMessage: m, highlightSource: null }),
  setHighlightSource: (n) => set({ highlightSource: n }),
}));
