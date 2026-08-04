export type User = {
  id: string;
  email: string;
  displayName: string;
};

export type LoginRequest = { email: string; password: string };
export type RegisterRequest = { email: string; password: string; displayName: string };
export type AuthResponse = { token: string; user: User };
