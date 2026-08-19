import {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
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

  useImperativeHandle(ref, () => innerRef.current as HTMLTextAreaElement);

  const resize = useCallback(() => {
    const el = innerRef.current;
    if (!el) return;
    el.style.height = 'auto';
    const max = autoSize?.maxRows ? autoSize.maxRows * 22 + 24 : undefined;
    const next = Math.min(el.scrollHeight, max ?? el.scrollHeight);
    el.style.height = `${next}px`;
  }, [autoSize]);

  useEffect(() => {
    resize();
  }, [resize, rest.value]);

  return (
    <textarea
      ref={(el) => {
        innerRef.current = el;
        if (typeof ref === 'function') ref(el);
      }}
      className={`${styles.textarea} ${className ?? ''}`.trim()}
      onInput={resize}
      {...rest}
    />
  );
});
