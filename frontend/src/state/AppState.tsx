import { createContext, useCallback, useContext, useEffect, useMemo, useState, useSyncExternalStore, type Dispatch, type SetStateAction, type ReactNode } from 'react';
import type { BusinessMembership, Queue } from '../api/types';
import { getMyBusinesses } from '../api/businesses';
import { APIError } from '../api/errors';
import { createAuthStore, type AuthState } from './authStore';
export type { AuthState } from './authStore';
export interface ActiveTicket { queue: Queue; joinedAs: 'guest' | 'user' | 'unknown' }
type AuthActions = Pick<ReturnType<typeof createAuthStore>, 'login' | 'register' | 'logout' | 'refreshUser'>;
interface AppState extends AuthActions {
  auth: AuthState;
  activeTicket: ActiveTicket | null; setActiveTicket: Dispatch<SetStateAction<ActiveTicket | null>>;
  selectedBusinessId: string | null; setSelectedBusinessId: Dispatch<SetStateAction<string | null>>;
  businesses: BusinessMembership[];
  businessesStatus: 'loading' | 'ready' | 'error';
  refreshBusinesses: (quiet?: boolean) => void;
}
const AppStateContext = createContext<AppState | null>(null);
export function AppStateProvider({ children }: { children: ReactNode }) {
  const [store] = useState(createAuthStore);
  const auth = useSyncExternalStore(store.subscribe, store.getSnapshot, store.getSnapshot);
  useEffect(() => { void store.refreshUser().catch(() => { /* Recoverable state is shown by the header. */ }); }, [store]);
  const [activeTicket, setActiveTicket] = useState<ActiveTicket | null>(null);
  const [selectedBusinessId, setSelectedBusinessId] = useState<string | null>(null);
  const [membership, setMembership] = useState<{ userId: string | null; items: BusinessMembership[]; status: 'loading' | 'ready' | 'error' }>({ userId: null, items: [], status: 'loading' });
  const [businessAttempt, setBusinessAttempt] = useState({ revision: 0, quiet: false });
  const refreshBusinesses = useCallback((quiet = false) => setBusinessAttempt(value => ({ revision: value.revision + 1, quiet })), []);
  const userId = auth.status === 'authenticated' ? auth.user.id : null;
  useEffect(() => { setSelectedBusinessId(null); }, [userId]);
  useEffect(() => {
    const controller = new AbortController();
    if (!userId) { setMembership({ userId: null, items: [], status: 'ready' }); return () => controller.abort(); }
    setMembership(previous => businessAttempt.quiet && previous.userId === userId && previous.status === 'ready' ? previous : { userId, items: [], status: 'loading' });
    void getMyBusinesses({ signal: controller.signal }).then(items => {
      if (controller.signal.aborted) return;
      setMembership({ userId, items, status: 'ready' });
    }).catch(error => {
      if (controller.signal.aborted) return;
      setMembership({ userId, items: [], status: 'error' });
      if (error instanceof APIError && error.status === 401) void store.refreshUser(true).catch(() => {});
    });
    return () => controller.abort();
  }, [userId, businessAttempt, store]);
  const businesses = membership.userId === userId ? membership.items : [];
  const businessesStatus = membership.userId === userId ? membership.status : 'loading';
  const value = useMemo(() => ({ auth, login: store.login, register: store.register, logout: store.logout, refreshUser: store.refreshUser, activeTicket, setActiveTicket, selectedBusinessId, setSelectedBusinessId, businesses, businessesStatus, refreshBusinesses }), [auth, store, activeTicket, selectedBusinessId, businesses, businessesStatus, refreshBusinesses]);
  return <AppStateContext.Provider value={value}>{children}</AppStateContext.Provider>;
}
export function useAppState() {
  const state = useContext(AppStateContext);
  if (!state) throw new Error('useAppState requires AppStateProvider');
  return state;
}
