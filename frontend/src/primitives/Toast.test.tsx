import { describe, it, expect } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToastProvider, useToast } from './ToastProvider';

function Harness() {
  const toast = useToast();
  return <button onClick={() => toast.error('出错了')}>trigger</button>;
}

describe('ToastProvider', () => {
  it('调用 toast.error 后渲染提示并自动消失', async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <Harness />
      </ToastProvider>,
    );
    await user.click(screen.getByRole('button', { name: 'trigger' }));
    expect(screen.getByText('出错了')).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText('出错了')).not.toBeInTheDocument(), {
      timeout: 5000,
    });
  });
});
