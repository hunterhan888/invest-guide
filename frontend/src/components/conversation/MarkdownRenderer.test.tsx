import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MarkdownRenderer } from './MarkdownRenderer';
import { linkSourceRefs } from '@/utils/sourceRefs';

describe('MarkdownRenderer', () => {
  it('linkSourceRefs 把片段[N]转义为指向来源的数字链接', () => {
    expect(linkSourceRefs('参考片段[1]与片段[2]内容')).toBe('参考[1](#src-1)与[2](#src-2)内容');
    expect(linkSourceRefs('无引用')).toBe('无引用');
  });

  it('linkSourceRefs 兼容【N】引用格式', () => {
    expect(linkSourceRefs('建议加强防范。【3】')).toBe('建议加强防范。[3](#src-3)');
    expect(linkSourceRefs('参考片段【1】')).toBe('参考[1](#src-1)');
  });

  it('linkSourceRefs 处理 "（来源：片段[N]）" 完整格式，并吃掉前置换行', () => {
    expect(linkSourceRefs('注意安全。（来源：片段[1]）')).toBe('注意安全。[1](#src-1)');
    expect(linkSourceRefs('做好评估。\n（来源：片段【2】）')).toBe('做好评估。[2](#src-2)');
  });

  it('渲染标题与列表', () => {
    const { container } = render(<MarkdownRenderer content={'# 标题\n- a\n- b'} />);
    expect(container.querySelector('h1')).not.toBeNull();
    expect(container.querySelectorAll('li')).toHaveLength(2);
  });

  it('不渲染内联 HTML / script', () => {
    const { container } = render(<MarkdownRenderer content={'<script>alert(1)</script>'} />);
    expect(container.querySelector('script')).toBeNull();
  });

  it('把「片段[N]」渲染为来源芯片', () => {
    const { container } = render(<MarkdownRenderer content={'参考片段[1]与片段[2]内容'} />);
    const chips = container.querySelectorAll('.source-chip');
    expect(chips).toHaveLength(2);
    expect(chips[0]).toHaveTextContent('1');
    expect(chips[1]).toHaveTextContent('2');
  });

  it('点击来源芯片触发 onSourceRef', () => {
    const onSourceRef = vi.fn();
    render(<MarkdownRenderer content={'参考片段[3]'} onSourceRef={onSourceRef} />);
    fireEvent.click(screen.getByText('3'));
    expect(onSourceRef).toHaveBeenCalledWith(3);
  });

  it('传入 sources 时芯片可渲染', () => {
    const sources = [{ id: 'c1', title: '测试来源', snippet: '这是测试片段内容' }];
    render(<MarkdownRenderer content={'参考片段[1]'} sources={sources} />);
    expect(screen.getByText('1')).toBeInTheDocument();
  });
});
