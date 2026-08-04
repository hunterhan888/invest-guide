import { App as AntdApp, ConfigProvider, theme } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { RouterProvider } from 'react-router-dom';
import { SWRConfig } from 'swr';
import './i18n/config';
import { router } from './router';
import { ErrorBoundary } from './components/ErrorBoundary';

export default function App() {
  const configProps = {
    theme: {
      algorithm: theme.defaultAlgorithm,
      cssVar: true,
      components: {
        Layout: {
          bodyBg: '#f5f8ff',
          footerBg: '#f5f8ff',
          headerBg: '#ffffff',
          headerColor: 'rgba(0, 0, 0, 0.88)',
          siderBg: '#ffffff',
          triggerBg: '#f0f5ff',
          triggerColor: 'rgba(0, 0, 0, 0.88)',
        },
        Menu: {
          activeBarBorderWidth: 0,
          itemBg: 'transparent',
          subMenuItemBg: 'transparent',
        },
        Button: {},
        Alert: {},
        Modal: {},
        Card: {},
        Tooltip: {},
        Checkbox: {},
        Radio: {},
        Select: {},
        Input: {},
        Switch: {},
        Progress: {
          circleTextColor: 'rgba(0, 0, 0, 0.88)',
          defaultColor: '#1677FF',
          remainingColor: 'rgba(0, 0, 0, 0.06)',
        },
        Steps: {},
        Slider: {},
        ColorPicker: {},
        Notification: {},
      },
    },
  };

  return (
    <ConfigProvider locale={zhCN} {...configProps}>
      <AntdApp>
        <ErrorBoundary>
          <SWRConfig
            value={{
              onError: (err) => {
                // 401 已在 api/client 层统一处理（logout + 跳登录）；此处仅兜底其他错误
                if (err?.status !== 401 && import.meta.env.DEV) {
                  console.error('SWR error', err);
                }
              },
            }}
          >
            <RouterProvider router={router} />
          </SWRConfig>
        </ErrorBoundary>
      </AntdApp>
    </ConfigProvider>
  );
}
