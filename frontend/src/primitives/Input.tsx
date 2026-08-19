import { forwardRef, type InputHTMLAttributes, type ReactNode } from 'react';
import styles from './Input.module.css';

export type InputProps = {
  icon?: ReactNode;
  className?: string;
} & Omit<InputHTMLAttributes<HTMLInputElement>, 'className'>;

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { icon, className, ...rest },
  ref,
) {
  return (
    <span className={styles.wrap}>
      {icon && <span className={styles.icon}>{icon}</span>}
      <input ref={ref} className={`${styles.input} ${className ?? ''}`.trim()} {...rest} />
    </span>
  );
});
