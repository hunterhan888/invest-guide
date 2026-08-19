import { type ReactNode } from 'react';
import styles from './Tooltip.module.css';

export function Tooltip({ content, children }: { content: ReactNode; children: ReactNode }) {
  return (
    <span className={styles.root}>
      {children}
      <span className={styles.bubble}>{content}</span>
    </span>
  );
}
