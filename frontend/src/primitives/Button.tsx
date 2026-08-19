import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react';
import styles from './Button.module.css';

type Variant = 'primary' | 'ghost' | 'outline' | 'toolbar';
type Size = 'sm' | 'md';

export type ButtonProps = {
  variant?: Variant;
  size?: Size;
  block?: boolean;
  icon?: ReactNode;
  loading?: boolean;
  danger?: boolean;
  className?: string;
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'className'>;

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'ghost', size = 'md', block, icon, loading, danger, className, children, disabled, ...rest },
  ref,
) {
  const classes = [
    styles.button,
    styles[variant],
    styles[size],
    block ? styles.block : '',
    danger ? styles.danger : '',
    className ?? '',
  ]
    .filter(Boolean)
    .join(' ');
  return (
    <button ref={ref} className={classes} disabled={disabled || loading} {...rest}>
      {icon && <span className={styles.icon}>{icon}</span>}
      {children != null && <span>{children}</span>}
      {loading && <span className={styles.spinner} aria-hidden="true" />}
    </button>
  );
});
