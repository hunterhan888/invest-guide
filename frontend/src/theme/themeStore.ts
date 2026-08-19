import { create } from 'zustand';

export type ThemeMode = 'light' | 'dark';

const THEME_KEY = 'investguide.theme';

type ThemeState = {
  mode: ThemeMode;
  setMode: (m: ThemeMode) => void;
  toggle: () => void;
  hydrate: () => void;
};

function readSaved(): ThemeMode {
  if (typeof window === 'undefined') return 'light';
  const saved = localStorage.getItem(THEME_KEY);
  if (saved === 'light' || saved === 'dark') return saved;
  if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }
  return 'light';
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  mode: 'light',
  setMode: (mode) => {
    localStorage.setItem(THEME_KEY, mode);
    set({ mode });
  },
  toggle: () => {
    const next = get().mode === 'dark' ? 'light' : 'dark';
    get().setMode(next);
  },
  hydrate: () => set({ mode: readSaved() }),
}));

useThemeStore.getState().hydrate();
