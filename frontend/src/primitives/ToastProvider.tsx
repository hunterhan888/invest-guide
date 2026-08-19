import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { Toast, type ToastKind } from './Toast';

type ToastItem = { id: number; kind: ToastKind; text: string };
type ToastApi = { success: (text: string) => void; error: (text: string) => void; info: (text: string) => void };

const ToastContext = createContext<ToastApi | null>(null);

const HOLD_MS = 3000;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const show = useCallback((kind: ToastKind, text: string) => {
    const id = ++nextId.current;
    setItems((cur) => [...cur, { id, kind, text }]);
    setTimeout(() => {
      setItems((cur) => cur.filter((t) => t.id !== id));
    }, HOLD_MS + 1000);
  }, []);

  const api = useMemo<ToastApi>(
    () => ({
      success: (t) => show('success', t),
      error: (t) => show('error', t),
      info: (t) => show('info', t),
    }),
    [show],
  );

  return (
    <ToastContext.Provider value={api}>
      {children}
      {items.map((t) => (
        <Toast key={t.id} kind={t.kind} text={t.text} />
      ))}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
}
