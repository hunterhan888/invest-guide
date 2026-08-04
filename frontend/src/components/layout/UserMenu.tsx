import { Avatar, Button, Dropdown, Typography } from 'antd';
import { LogoutOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';

export default function UserMenu() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  return (
    <Dropdown
      menu={{
        items: [
          {
            key: 'logout',
            icon: <LogoutOutlined />,
            label: t('sidebar.userMenu.logout'),
            onClick: () => {
              logout();
              navigate('/login', { replace: true });
            },
          },
        ],
      }}
    >
      <Button type="text" className="w-full flex items-center gap-2 !text-[20px]">
        <Avatar
          size="small"
          style={{ background: 'linear-gradient(135deg, #4096ff, #1677ff)', fontSize: 14 }}
        >
          {(user?.displayName ?? '?').slice(0, 1)}
        </Avatar>
        <Typography.Text className="truncate !text-[20px]">{user?.displayName}</Typography.Text>
      </Button>
    </Dropdown>
  );
}
