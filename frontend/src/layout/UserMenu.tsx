import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Dropdown } from '@/primitives/Dropdown';
import { LogoutIcon, MoonIcon, SunIcon } from '@/primitives/icons';
import { useAuthStore } from '@/stores/authStore';
import { useThemeStore } from '@/theme/themeStore';
import styles from './UserMenu.module.css';

export default function UserMenu() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const mode = useThemeStore((s) => s.mode);
  const toggleTheme = useThemeStore((s) => s.toggle);

  return (
    <Dropdown
      align="end"
      trigger={
        <button type="button" className={styles.trigger}>
          <span className={styles.avatar}>{(user?.displayName ?? '?').slice(0, 1)}</span>
          <span className={styles.name}>{user?.displayName}</span>
        </button>
      }
      items={[
        {
          key: 'theme',
          icon: mode === 'dark' ? <SunIcon size={14} /> : <MoonIcon size={14} />,
          label: t('sidebar.userMenu.theme'),
          onClick: toggleTheme,
        },
        {
          key: 'logout',
          icon: <LogoutIcon size={14} />,
          label: t('sidebar.userMenu.logout'),
          danger: true,
          onClick: () => {
            logout();
            navigate('/login', { replace: true });
          },
        },
      ]}
    />
  );
}
