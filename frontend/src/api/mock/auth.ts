import { registerMock, type MockRequest } from '../client';
import { jsonOk, jsonFail } from './tools';
import type { AuthResponse, User } from '../auth/types';

const users = new Map<string, { password: string; user: User; token: string }>();

function newId() {
  return Math.random().toString(36).slice(2);
}

export function __resetAuthData() {
  users.clear();
}

export function installAuthMocks() {
  registerMock({
    match: (r) => r.method === 'POST' && r.path === '/auth/register',
    handle: async (r: MockRequest) => {
      const { email, password, displayName } = r.body as {
        email: string;
        password: string;
        displayName: string;
      };
      if (users.has(email)) return jsonFail('CONFLICT', 'email exists', 409);
      const user: User = { id: newId(), email, displayName };
      const token = 'tok_' + newId();
      users.set(email, { password, user, token });
      return jsonOk<AuthResponse>({ token, user }, 201);
    },
  });

  registerMock({
    match: (r) => r.method === 'POST' && r.path === '/auth/login',
    handle: async (r: MockRequest) => {
      const { email, password } = r.body as { email: string; password: string };
      const rec = users.get(email);
      if (!rec || rec.password !== password) {
        return jsonFail('UNAUTHORIZED', 'invalid credentials', 401);
      }
      return jsonOk<AuthResponse>({ token: rec.token, user: rec.user });
    },
  });
}
