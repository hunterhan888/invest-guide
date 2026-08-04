import { Button, Layout } from 'antd';
import { MenuFoldOutlined, MenuUnfoldOutlined } from '@ant-design/icons';
import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import UserMenu from './UserMenu';
import { useUiStore } from '@/stores/uiStore';

const { Sider, Content } = Layout;

export default function AppLayout() {
  const collapsed = useUiStore((s) => s.sidebarCollapsed);
  const toggle = useUiStore((s) => s.toggleSidebar);

  return (
    <Layout className="h-screen">
      <Sider
        theme="light"
        width={340}
        collapsedWidth={64}
        collapsed={collapsed}
        trigger={null}
        className="border-r border-border bg-white"
      >
        <div className="flex h-full flex-col">
          <div className="min-h-0 flex-1 overflow-hidden">
            <Sidebar />
          </div>
          <div className="shrink-0 border-t border-border p-2">
            <UserMenu />
          </div>
          <div className="shrink-0 border-t border-border">
            <Button
              type="text"
              block
              className="!h-10 flex items-center justify-center text-fg-tertiary"
              icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              onClick={toggle}
            />
          </div>
        </div>
      </Sider>
      <Layout className="bg-bg-layout">
        <Content className="overflow-hidden">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
