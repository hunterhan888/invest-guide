import type { MockResponse } from '../client';

export function jsonOk<T>(data: T, status = 200): MockResponse {
  return { status, body: { success: true, data } };
}

export function jsonFail(code: string, message: string, status: number): MockResponse {
  return { status, body: { success: false, error: message, code } };
}

export function delay(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

export type SSEFrame = { event: string; data: unknown };

export function sseStream(frames: SSEFrame[]): MockResponse {
  const encoder = new TextEncoder();
  let cancelled = false;
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      void (async () => {
        for (const f of frames) {
          if (cancelled) return;
          const payload = `event: ${f.event}\ndata: ${JSON.stringify(f.data)}\n\n`;
          controller.enqueue(encoder.encode(payload));
          await delay(20);
        }
        if (!cancelled) controller.close();
      })();
    },
    cancel() {
      cancelled = true;
    },
  });
  return { status: 200, stream };
}
