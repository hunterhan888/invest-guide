import { create } from 'zustand';

type UiState = {
  sidebarCollapsed: boolean;
  detailsOpen: boolean;
  sidebarWidth: number;
  detailsWidth: number;
  toggleSidebar: () => void;
  setCollapsed: (v: boolean) => void;
  setDetailsOpen: (v: boolean) => void;
  setSidebarWidth: (v: number) => void;
  setDetailsWidth: (v: number) => void;
};

const SIDEBAR_DEFAULT = 340;
const DETAILS_DEFAULT = 360;

function readNum(key: string, fallback: number): number {
  const raw = localStorage.getItem(key);
  const n = raw ? Number(raw) : NaN;
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

export const useUiStore = create<UiState>((set, get) => ({
  sidebarCollapsed: localStorage.getItem('investguide.sidebarCollapsed') === 'true',
  detailsOpen: localStorage.getItem('investguide.detailsOpen') === 'true',
  sidebarWidth: readNum('investguide.sidebarWidth', SIDEBAR_DEFAULT),
  detailsWidth: readNum('investguide.detailsWidth', DETAILS_DEFAULT),
  toggleSidebar: () => {
    const next = !get().sidebarCollapsed;
    localStorage.setItem('investguide.sidebarCollapsed', String(next));
    set({ sidebarCollapsed: next });
  },
  setCollapsed: (v) => {
    localStorage.setItem('investguide.sidebarCollapsed', String(v));
    set({ sidebarCollapsed: v });
  },
  setDetailsOpen: (v) => {
    localStorage.setItem('investguide.detailsOpen', String(v));
    set({ detailsOpen: v });
  },
  setSidebarWidth: (v) => {
    localStorage.setItem('investguide.sidebarWidth', String(v));
    set({ sidebarWidth: v });
  },
  setDetailsWidth: (v) => {
    localStorage.setItem('investguide.detailsWidth', String(v));
    set({ detailsWidth: v });
  },
}));
