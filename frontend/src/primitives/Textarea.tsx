import {
  forwardRef,
  useCallback,
  useEffect,
  useRef,
  type TextareaHTMLAttributes,
} from 'react';
import styles from './Textarea.module.css';

export type TextareaProps = {
  className?: string;
  autoSize?: { minRows?: number; maxRows?: number };
} & Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'className'>;

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className, autoSize, ...rest },
  ref,
) {
  const innerRef = useRef<HTMLTextAreaElement | null>(null);

  const resize = useCallback(() => {
    const el = innerRef.current;
    if (!el || el.scrollHeight === 0) return;
    el.style.height = 'auto';
    const lineH = 22;
    const min = autoSize?.minRows ? autoSize.minRows * lineH + 24 : undefined;
    const max = autoSize?.maxRows ? autoSize.maxRows * lineH + 24 : undefined;
    const height = Math.min(el.scrollHeight, max ?? el.scrollHeight);
    el.style.height = `${Math.max(height, min ?? height)}px`;
  }, [autoSize]);

  useEffect(() => {
    resize();
  }, [resize, rest.value]);

  return (
    <textarea
      ref={(el) => {
        innerRef.current = el;
        if (typeof ref === 'function') ref(el);
        else if (ref) ref.current = el;
      }}
      className={`${styles.textarea} ${className ?? ''}`.trim()}
      onInput={resize}
      {...rest}
    />
  );
});
