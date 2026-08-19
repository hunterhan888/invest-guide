import { useCallback, useRef, type ReactNode } from 'react';
import { useUiStore } from '@/stores/uiStore';
import styles from './AppFrame.module.css';

type AppFrameProps = {
  sidebar: ReactNode;
  main: ReactNode;
  details?: ReactNode;
};

export function AppFrame({ sidebar, main, details }: AppFrameProps) {
  const sidebarCollapsed = useUiStore((s) => s.sidebarCollapsed);
  const detailsOpen = useUiStore((s) => s.detailsOpen);
  const sidebarWidth = useUiStore((s) => s.sidebarWidth);
  const detailsWidth = useUiStore((s) => s.detailsWidth);
  const setSidebarWidth = useUiStore((s) => s.setSidebarWidth);
  const setDetailsWidth = useUiStore((s) => s.setDetailsWidth);

  const draggingRef = useRef<'sidebar' | 'details' | null>(null);

  const onPointerDown = useCallback(
    (side: 'sidebar' | 'details') => (e: React.PointerEvent) => {
      e.preventDefault();
      draggingRef.current = side;
      const startX = e.clientX;
      const startWidth = side === 'sidebar' ? sidebarWidth : detailsWidth;
      const setter = side === 'sidebar' ? setSidebarWidth : setDetailsWidth;

      function onMove(ev: PointerEvent) {
        const delta = ev.clientX - startX;
        const next = side === 'sidebar' ? startWidth + delta : startWidth - delta;
        setter(Math.min(Math.max(next, 200), 560));
      }
      function onUp() {
        draggingRef.current = null;
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
      }
      window.addEventListener('pointermove', onMove);
      window.addEventListener('pointerup', onUp);
    },
    [sidebarWidth, detailsWidth, setSidebarWidth, setDetailsWidth],
  );

  return (
    <div
      className={styles.frame}
      data-dragging={draggingRef.current != null}
      style={{
        gridTemplateColumns: `${sidebarCollapsed ? 56 : sidebarWidth}px minmax(0, 1fr) ${
          detailsOpen ? detailsWidth : 0
        }px`,
      }}
    >
      <aside className={styles.sidebarCol}>{sidebar}</aside>
      {!sidebarCollapsed && (
        <div
          className={styles.handle}
          data-side="sidebar"
          role="separator"
          aria-orientation="vertical"
          style={{ left: `${sidebarWidth}px` }}
          onPointerDown={onPointerDown('sidebar')}
        />
      )}
      <main className={styles.centerCol}>{main}</main>
      {detailsOpen && details && (
        <div
          className={styles.handle}
          data-side="details"
          role="separator"
          aria-orientation="vertical"
          style={{ left: `calc(100% - ${detailsWidth}px)` }}
          onPointerDown={onPointerDown('details')}
        />
      )}
      <section className={styles.detailsCol} data-collapsed={!detailsOpen || undefined}>
        {details}
      </section>
    </div>
  );
}
