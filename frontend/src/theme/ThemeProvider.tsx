import { useEffect, type ReactNode } from 'react';
import { useThemeStore } from './themeStore';

export function ThemeProvider({ children }: { children: ReactNode }) {
  const mode = useThemeStore((s) => s.mode);

  useEffect(() => {
    document.body.toggleAttribute('data-ds-dark-theme', mode === 'dark');
  }, [mode]);

  return <>{children}</>;
}
