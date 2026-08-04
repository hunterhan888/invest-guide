import { create } from 'zustand';
import type { User } from '@/api/auth/types';

export const TOKEN_KEY = 'investguide.token';
export const SESSION_TOKEN_KEY = 'investguide.token.session';

type AuthState = {
  token: string | null;
  user: User | null;
  login: (p: { token: string; user: User; remember?: boolean }) => void;
  setUser: (u: User) => void;
  logout: () => void;
  hydrate: () => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  login: ({ token, user, remember }) => {
    if (remember === false) {
      sessionStorage.setItem(SESSION_TOKEN_KEY, token);
      localStorage.removeItem(TOKEN_KEY);
    } else {
      localStorage.setItem(TOKEN_KEY, token);
      sessionStorage.removeItem(SESSION_TOKEN_KEY);
    }
    set({ token, user });
  },
  setUser: (user) => set({ user }),
  logout: () => {
    localStorage.removeItem(TOKEN_KEY);
    sessionStorage.removeItem(SESSION_TOKEN_KEY);
    set({ token: null, user: null });
  },
  hydrate: () => {
    const token = localStorage.getItem(TOKEN_KEY) ?? sessionStorage.getItem(SESSION_TOKEN_KEY);
    if (token) set({ token });
  },
}));

useAuthStore.getState().hydrate();
