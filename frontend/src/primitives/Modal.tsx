import { useEffect, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import styles from './Modal.module.css';
import { CloseIcon } from './icons';

export type ModalProps = {
  open: boolean;
  onClose?: () => void;
  title?: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
};

export function Modal({ open, onClose, title, description, children, footer }: ModalProps) {
  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose?.();
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div className={styles.root} role="presentation">
      <div className={styles.mask} onClick={onClose} data-testid="modal-mask" />
      <div className={styles.dialog} role="dialog" aria-modal="true">
        <div className={styles.header}>
          {title && <h3 className={styles.title}>{title}</h3>}
          {onClose && (
            <button type="button" className={styles.close} onClick={onClose} aria-label="close">
              <CloseIcon />
            </button>
          )}
        </div>
        {description && <p className={styles.description}>{description}</p>}
        {children && <div className={styles.body}>{children}</div>}
        {footer && <div className={styles.footer}>{footer}</div>}
      </div>
    </div>,
    document.body,
  );
}
