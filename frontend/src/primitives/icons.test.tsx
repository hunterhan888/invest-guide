import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { SendIcon } from './icons';

describe('icons', () => {
  it('渲染 SVG 并支持 size', () => {
    const { container } = render(<SendIcon size={20} />);
    const svg = container.querySelector('svg');
    expect(svg).not.toBeNull();
    expect(svg).toHaveAttribute('width', '20');
  });
});
