import { useId } from 'react';

export function CompassLogo({ size = 28 }: { size?: number }) {
  const id = useId();
  const gradId = `ig-logo-grad-${id}`;
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <circle cx="16" cy="16" r="15" fill={`url(#${gradId})`} />
      <circle cx="16" cy="16" r="15" stroke="rgba(255,255,255,0.35)" strokeWidth="1" />
      <path d="M21.5 10.5l-2.6 7.8-7.8 2.6 2.6-7.8z" fill="#fff" />
      <circle cx="16" cy="16" r="2.2" fill="#fff" />
      <defs>
        <linearGradient id={gradId} x1="0" y1="0" x2="32" y2="32">
          <stop stopColor="#6792f5" />
          <stop offset="1" stopColor="#3f72d8" />
        </linearGradient>
      </defs>
    </svg>
  );
}
