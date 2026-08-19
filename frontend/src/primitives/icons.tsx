import type { ReactNode } from 'react';

type IconProps = { size?: number; className?: string };
type BaseProps = IconProps & { children: ReactNode };

function Base({ size = 16, className, children }: BaseProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      {children}
    </svg>
  );
}

export function SendIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 19V5" />
      <path d="M5 12l7-7 7 7" />
    </Base>
  );
}

export function StopIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="7" y="7" width="10" height="10" rx="2" />
    </Base>
  );
}

export function PlusIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 5v14M5 12h14" />
    </Base>
  );
}

export function MoreIcon(p: IconProps) {
  return (
    <Base {...p}>
      <circle cx="5" cy="12" r="1.6" fill="currentColor" stroke="none" />
      <circle cx="12" cy="12" r="1.6" fill="currentColor" stroke="none" />
      <circle cx="19" cy="12" r="1.6" fill="currentColor" stroke="none" />
    </Base>
  );
}

export function CloseIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M18 6L6 18M6 6l12 12" />
    </Base>
  );
}

export function LogoutIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <path d="M16 17l5-5-5-5" />
      <path d="M21 12H9" />
    </Base>
  );
}

export function SunIcon(p: IconProps) {
  return (
    <Base {...p}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </Base>
  );
}

export function MoonIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
    </Base>
  );
}

export function PanelLeftIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M9 3v18" />
    </Base>
  );
}

export function PanelRightIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M15 3v18" />
    </Base>
  );
}

export function ChevronDownIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M6 9l6 6 6-6" />
    </Base>
  );
}

export function ChevronRightIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M9 6l6 6-6 6" />
    </Base>
  );
}

export function ArrowDownIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 5v14M5 12l7 7 7-7" />
    </Base>
  );
}

export function DeleteIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
      <path d="M10 11v6M14 11v6" />
    </Base>
  );
}

export function CopyIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="9" y="9" width="13" height="13" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </Base>
  );
}

export function RefreshIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M21 12a9 9 0 1 1-2.6-6.4M21 3v6h-6" />
    </Base>
  );
}

export function DocumentIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <path d="M14 2v6h6" />
    </Base>
  );
}

export function ClockIcon(p: IconProps) {
  return (
    <Base {...p}>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 3" />
    </Base>
  );
}

export function SparkleIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" />
    </Base>
  );
}

export function ChevronsUpIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M17 11l-5-5-5 5M17 18l-5-5-5 5" />
    </Base>
  );
}
