import { type ReactNode } from 'react';
import styles from './fields.module.css';

export function Field({ label, error, children }: { label: ReactNode; error?: string; children: ReactNode }) {
  return (
    <label className={styles.field}>
      <span className={styles.label}>{label}</span>
      {children}
      {error && <span className={styles.error}>{error}</span>}
    </label>
  );
}
