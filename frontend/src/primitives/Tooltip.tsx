import { type ReactNode } from 'react';
import styles from './Tooltip.module.css';

export function Tooltip({ content, children }: { content: ReactNode; children: ReactNode }) {
  return (
    <span className={styles.root} data-tooltip={typeof content === 'string' ? content : undefined}>
      {children}
      {typeof content !== 'string' && <span className={styles.bubble}>{content}</span>}
    </span>
  );
}
