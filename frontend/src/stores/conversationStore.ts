import { create } from 'zustand';

type ConversationState = {
  activeId: string | null;
  setActive: (id: string | null) => void;
  clearActive: () => void;
};

export const useConversationStore = create<ConversationState>((set) => ({
  activeId: null,
  setActive: (id) => set({ activeId: id }),
  clearActive: () => set({ activeId: null }),
}));
