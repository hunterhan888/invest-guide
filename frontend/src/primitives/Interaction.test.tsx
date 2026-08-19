import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Dropdown } from './Dropdown';
import { Tooltip } from './Tooltip';

describe('Dropdown', () => {
  it('点击触发项展开菜单，选中项触发 onSelect', async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    render(
      <Dropdown
        trigger={<button>菜单</button>}
        items={[{ key: 'a', label: '选项A', onClick: () => onSelect('a') }]}
      />,
    );
    await user.click(screen.getByRole('button', { name: '菜单' }));
    await user.click(screen.getByText('选项A'));
    expect(onSelect).toHaveBeenCalledWith('a');
  });
});

describe('Tooltip', () => {
  it('渲染 tooltip 文本', () => {
    render(<Tooltip content="提示内容">hover</Tooltip>);
    expect(screen.getByText('提示内容')).toBeInTheDocument();
  });
});
