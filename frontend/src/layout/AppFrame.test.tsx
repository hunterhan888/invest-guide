import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AppFrame } from './AppFrame';

describe('AppFrame', () => {
  it('渲染侧栏/主区/详情三栏内容', () => {
    render(
      <AppFrame sidebar={<div>Sidebar</div>} main={<div>Main</div>} details={<div>Details</div>} />,
    );
    expect(screen.getByText('Sidebar')).toBeInTheDocument();
    expect(screen.getByText('Main')).toBeInTheDocument();
    expect(screen.getByText('Details')).toBeInTheDocument();
  });
});
