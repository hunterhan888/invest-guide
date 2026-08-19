import { Outlet } from 'react-router-dom';
import { AppFrame } from './AppFrame';
import Sidebar from './Sidebar';
import { DetailsPanel } from './DetailsPanel';
import UserMenu from './UserMenu';
import styles from './AppLayout.module.css';

export default function AppLayout() {
  return (
    <AppFrame
      sidebar={
        <>
          <div className={styles.sidebarSeat}>
            <Sidebar />
          </div>
          <div className={styles.userMenuSeat}>
            <UserMenu />
          </div>
        </>
      }
      main={<Outlet />}
      details={<DetailsPanel />}
    />
  );
}
