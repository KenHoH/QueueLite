import * as authAPI from '../api/auth';
import { APIError } from '../api/errors';
import type { LoginRequest, RegisterUserRequest, User } from '../api/types';

export type AuthState = { status: 'loading' | 'guest'; user: null } | { status: 'authenticated'; user: User } | { status: 'error'; user: User | null };
export class RegistrationSignInError extends Error {
  constructor() { super('Your account was created, but sign-in could not be completed. Please sign in to continue.'); }
}
/** In-memory state only. The server cookie is always the session authority. */
export function createAuthStore(service = authAPI) {
  let state: AuthState = { status: 'loading', user: null };
  let revision = 0;
  let pending = false;
  const listeners = new Set<() => void>();
  const publish = (next: AuthState) => { state = next; listeners.forEach(listener => listener()); };
  const load = async (current: number) => {
    try {
      const user = await service.getCurrentUser();
      if (current === revision) publish({ status: 'authenticated', user });
      return user;
    } catch (error) {
      if (current === revision) publish(error instanceof APIError && error.status === 401
        ? { status: 'guest', user: null } : { status: 'error', user: state.user });
      throw error;
    }
  };
  const mutate = async (action: (current: number) => Promise<void>) => {
    if (pending) throw new Error('Authentication request already in progress.');
    pending = true;
    const current = ++revision; // A stale bootstrap response must not overwrite a mutation.
    try { await action(current); }
    catch (error) {
      // A failed mutation can supersede startup; never leave that session check spinning.
      if (state.status === 'loading') publish({ status: 'error', user: null });
      throw error;
    } finally { pending = false; }
  };
  return {
    getSnapshot: () => state,
    subscribe: (listener: () => void) => { listeners.add(listener); return () => { listeners.delete(listener); }; },
    refreshUser: async (quiet = false) => {
      if (pending) return;
      const current = ++revision;
      if (!quiet || state.status !== 'authenticated') publish({ status: 'loading', user: null });
      await load(current);
    },
    login: (credentials: LoginRequest) => mutate(async current => {
      await service.login(credentials);
      await load(current);
    }),
    register: (credentials: RegisterUserRequest) => mutate(async current => {
      await service.registerUser(credentials);
      try {
        await service.login({ username: credentials.username, password: credentials.password });
        await load(current);
      } catch { throw new RegistrationSignInError(); }
    }),
    logout: () => mutate(async () => {
      await service.logout();
      publish({ status: 'guest', user: null });
    }),
  };
}
