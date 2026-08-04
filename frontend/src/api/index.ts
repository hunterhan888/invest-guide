import { installMocks } from './mock';

if (import.meta.env.VITE_USE_MOCK === 'true') {
  installMocks();
}

export { request, openStream } from './client';
export * from './types';
