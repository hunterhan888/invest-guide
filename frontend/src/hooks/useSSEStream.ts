import { useEffect, useRef, useState } from 'react';
import { openStream } from '@/api/client';
import { parseFrames } from './sseParser';

export type SSEEvent =
  | { type: 'heartbeat' }
  | { type: 'sources'; chunks: { id: string; title: string; snippet: string }[] }
  | { type: 'message'; delta: string }
  | { type: 'done'; messageId: string; tokensUsed: number }
  | { type: 'error'; code: string; message: string };

type Opts = {
  convId: string;
  messageId: string;
  enabled?: boolean;
  onEvent: (e: SSEEvent) => void;
};

const HEARTBEAT_TIMEOUT_MS = 35_000;
const MAX_RECONNECT_ATTEMPTS = 3;

export function useSSEStream({ convId, messageId, enabled = true, onEvent }: Opts) {
  const [state, setState] = useState<'idle' | 'streaming' | 'done' | 'error'>('idle');
  const abortRef = useRef<AbortController | null>(null);
  const lastEventIdRef = useRef<string | undefined>(undefined);
  const onEventRef = useRef(onEvent);
  const stoppedRef = useRef(false);
  const timersRef = useRef<{
    heartbeat: ReturnType<typeof setTimeout> | null;
    retry: ReturnType<typeof setTimeout> | null;
  }>({ heartbeat: null, retry: null });
  onEventRef.current = onEvent;

  useEffect(() => {
    if (!enabled) return;
    let reconnectAttempts = 0;
    stoppedRef.current = false;

    function clearTimers() {
      if (timersRef.current.heartbeat) clearTimeout(timersRef.current.heartbeat);
      if (timersRef.current.retry) clearTimeout(timersRef.current.retry);
      timersRef.current.heartbeat = null;
      timersRef.current.retry = null;
    }

    function resetHeartbeat() {
      if (timersRef.current.heartbeat) clearTimeout(timersRef.current.heartbeat);
      timersRef.current.heartbeat = setTimeout(() => {
        abortRef.current?.abort();
        scheduleReconnect();
      }, HEARTBEAT_TIMEOUT_MS);
    }

    function scheduleReconnect() {
      if (stoppedRef.current) return;
      if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
        setState('error');
        onEventRef.current({ type: 'error', code: 'STREAM_ERROR', message: 'stream unavailable' });
        return;
      }
      const delay = Math.pow(2, reconnectAttempts) * 1000;
      reconnectAttempts++;
      timersRef.current.retry = setTimeout(() => {
        if (!stoppedRef.current) void start();
      }, delay);
    }

    async function start() {
      const abort = new AbortController();
      abortRef.current = abort;
      try {
        const stream = await openStream(
          `/conversations/${convId}/messages/${messageId}/stream`,
          lastEventIdRef.current,
        );
        if (stoppedRef.current) return;
        setState('streaming');
        resetHeartbeat();
        const reader = stream.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        for (;;) {
          if (stoppedRef.current) {
            reader.cancel();
            return;
          }
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const { events, rest } = parseFrames(buffer);
          buffer = rest;
          for (const ev of events) {
            if (ev.id) lastEventIdRef.current = ev.id;
            let parsed: SSEEvent;
            try {
              parsed = parseEvent(ev.event, ev.data);
            } catch {
              continue;
            }
            if (parsed.type === 'heartbeat') {
              resetHeartbeat();
              continue;
            }
            if (!stoppedRef.current) {
              onEventRef.current(parsed);
            }
            resetHeartbeat();
            if (parsed.type === 'done' || parsed.type === 'error') {
              setState(parsed.type);
              clearTimers();
              return;
            }
          }
        }
      } catch {
        if (stoppedRef.current || abort.signal.aborted) return;
        scheduleReconnect();
      }
    }

    void start();
    return () => {
      stoppedRef.current = true;
      clearTimers();
      abortRef.current?.abort();
    };
  }, [convId, messageId, enabled]);

  function stop() {
    stoppedRef.current = true;
    if (timersRef.current.heartbeat) clearTimeout(timersRef.current.heartbeat);
    if (timersRef.current.retry) clearTimeout(timersRef.current.retry);
    abortRef.current?.abort();
    setState('idle');
  }

  return { state, stop };
}

function parseEvent(event: string, data: string): SSEEvent {
  const obj = JSON.parse(data);
  switch (event) {
    case 'heartbeat':
      return { type: 'heartbeat' };
    case 'sources':
      return { type: 'sources', chunks: obj.chunks };
    case 'message':
      return { type: 'message', delta: obj.delta };
    case 'done':
      return { type: 'done', messageId: obj.messageId, tokensUsed: obj.tokensUsed };
    case 'error':
      return { type: 'error', code: obj.code, message: obj.message };
    default:
      return { type: 'heartbeat' };
  }
}
