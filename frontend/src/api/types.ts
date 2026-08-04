export type ErrorCode =
  | 'INVALID_INPUT'
  | 'UNAUTHORIZED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'RATE_LIMITED'
  | 'INTERNAL_ERROR'
  | 'BAD_GATEWAY'
  | 'GATEWAY_TIMEOUT';

export type ApiResponse<T> =
  { success: true; data: T; message?: string } | { success: false; error: string; code: ErrorCode };

export type Paginated<T> = { items: T[]; total: number; hasMore: boolean };

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: ErrorCode,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
