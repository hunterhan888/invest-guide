import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DisclosureRow } from './DisclosureRow';
import { Pill } from './Pill';

describe('DisclosureRow', () => {
  it('默认收起，点击展开内容', async () => {
    const user = userEvent.setup();
    render(
      <DisclosureRow title="引用来源">
        <div>来源内容</div>
      </DisclosureRow>,
    );
    expect(screen.queryByText('来源内容')).not.toBeInTheDocument();
    await user.click(screen.getByText('引用来源'));
    expect(screen.getByText('来源内容')).toBeInTheDocument();
  });

  it('controlled expanded 生效', () => {
    render(
      <DisclosureRow title="标题" expanded>
        <div>可见</div>
      </DisclosureRow>,
    );
    expect(screen.getByText('可见')).toBeInTheDocument();
  });
});

describe('Pill', () => {
  it('渲染文本', () => {
    render(<Pill>标签</Pill>);
    expect(screen.getByText('标签')).toBeInTheDocument();
  });
});
