import { RouterProvider } from 'react-router-dom';
import { SWRConfig } from 'swr';
import './i18n/config';
import { router } from './router';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ThemeProvider } from './theme/ThemeProvider';
import { ToastProvider } from './primitives/ToastProvider';

export default function App() {
  return (
    <ThemeProvider>
      <ToastProvider>
        <ErrorBoundary>
          <SWRConfig
            value={{
              onError: (err) => {
                if (err?.status !== 401 && import.meta.env.DEV) {
                  console.error('SWR error', err);
                }
              },
            }}
          >
            <RouterProvider router={router} />
          </SWRConfig>
        </ErrorBoundary>
      </ToastProvider>
    </ThemeProvider>
  );
}
