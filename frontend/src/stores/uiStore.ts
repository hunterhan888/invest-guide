import { create } from 'zustand';

type UiState = {
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
  setCollapsed: (v: boolean) => void;
};

export const useUiStore = create<UiState>((set, get) => ({
  sidebarCollapsed: localStorage.getItem('investguide.sidebarCollapsed') === 'true',
  toggleSidebar: () => {
    const next = !get().sidebarCollapsed;
    localStorage.setItem('investguide.sidebarCollapsed', String(next));
    set({ sidebarCollapsed: next });
  },
  setCollapsed: (v) => {
    localStorage.setItem('investguide.sidebarCollapsed', String(v));
    set({ sidebarCollapsed: v });
  },
}));
