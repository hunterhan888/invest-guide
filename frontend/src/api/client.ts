import { ApiError, type ApiResponse } from './types';
import { useAuthStore } from '@/stores/authStore';

export type MockRequest = { method: string; path: string; body: unknown; authToken?: string };
export type MockResponse =
  { status: number; body: unknown } | { status: number; stream: ReadableStream<Uint8Array> };
export type MockHandler = {
  match: (r: MockRequest) => boolean;
  handle: (r: MockRequest) => Promise<MockResponse>;
};

const handlers: MockHandler[] = [];
const USE_MOCK = import.meta.env.VITE_USE_MOCK === 'true';

export function registerMock(h: MockHandler) {
  handlers.push(h);
}

export function __resetMocks() {
  handlers.length = 0;
}

function readToken(): string | null {
  return useAuthStore.getState().token;
}

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  if (USE_MOCK) {
    const req: MockRequest = { method, path, body, authToken: readToken() ?? undefined };
    for (const h of handlers) {
      if (h.match(req)) {
        const res = await h.handle(req);
        if ('stream' in res) throw new Error('use openStream for SSE responses');
        return unwrap<T>(res.status, res.body as ApiResponse<T>);
      }
    }
    throw new ApiError(404, 'NOT_FOUND', `mock route not found: ${method} ${path}`);
  }

  const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(readToken() ? { Authorization: `Bearer ${readToken()}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (res.status === 204) return undefined as T;
  const json = (await res.json()) as ApiResponse<T>;
  const result = unwrap<T>(res.status, json);
  if (res.status === 401 && readToken()) handleUnauthorized();
  return result;
}

function handleUnauthorized() {
  const auth = useAuthStore.getState();
  if (auth.token) {
    auth.logout();
    if (window.location.pathname !== '/login') {
      window.location.assign('/login');
    }
  }
}

function unwrap<T>(status: number, json: ApiResponse<T>): T {
  if (json.success) return json.data;
  throw new ApiError(status, json.code, json.error);
}

export async function openStream(
  path: string,
  lastEventId?: string,
): Promise<ReadableStream<Uint8Array>> {
  if (USE_MOCK) {
    const req: MockRequest = {
      method: 'GET',
      path,
      body: undefined,
      authToken: readToken() ?? undefined,
    };
    for (const h of handlers) {
      if (h.match(req)) {
        const res = await h.handle(req);
        if ('stream' in res) return res.stream;
        throw new Error('expected stream response');
      }
    }
    throw new ApiError(404, 'NOT_FOUND', `mock stream not found: ${path}`);
  }

  const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}${path}`, {
    method: 'GET',
    headers: {
      ...(readToken() ? { Authorization: `Bearer ${readToken()}` } : {}),
      ...(lastEventId ? { 'Last-Event-ID': lastEventId } : {}),
    },
  });
  if (!res.ok || !res.body) {
    throw new ApiError(res.status, 'BAD_GATEWAY', 'stream open failed');
  }
  return res.body;
}
