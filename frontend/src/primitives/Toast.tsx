import styles from './Toast.module.css';

export type ToastKind = 'success' | 'error' | 'info';

const KIND_ICON: Record<ToastKind, string> = {
  success: '✓',
  error: '✕',
  info: 'i',
};

export function Toast({ kind, text }: { kind: ToastKind; text: string }) {
  return (
    <div className={styles.toast} role="status">
      <span className={`${styles.icon} ${styles[kind]}`} aria-hidden="true">
        {KIND_ICON[kind]}
      </span>
      <span className={styles.text}>{text}</span>
    </div>
  );
}
