import { useState, type ReactNode } from 'react';
import styles from './DisclosureRow.module.css';
import { ChevronDownIcon } from './icons';

type DisclosureRowProps = {
  title: ReactNode;
  children: ReactNode;
  defaultOpen?: boolean;
  expanded?: boolean;
  onToggle?: (open: boolean) => void;
};

export function DisclosureRow({ title, children, defaultOpen = false, expanded, onToggle }: DisclosureRowProps) {
  const [innerOpen, setInnerOpen] = useState(defaultOpen);
  const open = expanded ?? innerOpen;

  function toggle() {
    const next = !open;
    setInnerOpen(next);
    onToggle?.(next);
  }

  return (
    <div className={styles.root} data-open={open}>
      <button type="button" className={styles.summary} onClick={toggle} aria-expanded={open}>
        <span className={styles.title}>{title}</span>
        <span className={styles.chevron}>
          <ChevronDownIcon size={14} />
        </span>
      </button>
      {open && <div className={styles.body}>{children}</div>}
    </div>
  );
}
