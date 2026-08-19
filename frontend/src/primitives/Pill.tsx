import { type ReactNode } from 'react';
import styles from './Pill.module.css';

export function Pill({ children, tone }: { children: ReactNode; tone?: 'default' | 'accent' }) {
  return <span className={`${styles.pill} ${tone === 'accent' ? styles.accent : ''}`.trim()}>{children}</span>;
}
