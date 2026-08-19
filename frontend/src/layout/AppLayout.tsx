import { Outlet } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { PanelLeftIcon, PanelRightIcon } from '@/primitives/icons';
import { useUiStore } from '@/stores/uiStore';
import { AppFrame } from './AppFrame';
import Sidebar from './Sidebar';
import { DetailsPanel } from './DetailsPanel';
import UserMenu from './UserMenu';
import styles from './AppLayout.module.css';

export default function AppLayout() {
  const { t } = useTranslation();
  const collapsed = useUiStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useUiStore((s) => s.toggleSidebar);

  return (
    <AppFrame
      sidebar={
        <>
          <div className={styles.sidebarSeat}>
            <Sidebar collapsed={collapsed} />
          </div>
          <div className={styles.userMenuSeat}>
            <UserMenu />
            <Button
              variant="ghost"
              block
              className={styles.collapseBtn}
              icon={collapsed ? <PanelRightIcon size={14} /> : <PanelLeftIcon size={14} />}
              onClick={toggleSidebar}
              aria-label={t('sidebar.toggle')}
            >
              {collapsed ? '' : t('sidebar.toggle')}
            </Button>
          </div>
        </>
      }
      main={<Outlet />}
      details={<DetailsPanel />}
    />
  );
}
