import { useCallback, useEffect, useRef, useState } from 'react';
import { APIError } from '../api/errors';
import { useAppState } from '../state/AppState';

/** One cancellable read at a time. Poll only visible tabs; clear stale data on errors. */
export function useOperations<T>(load: (signal: AbortSignal) => Promise<T>, poll = true) {
  const { refreshUser } = useAppState();
  const [state, setState] = useState<{ data: T | null; loading: boolean; error: unknown }>({ data: null, loading: true, error: null });
  const [refresh, setRefresh] = useState(0);
  const controller = useRef<AbortController | null>(null);
  const retry = useCallback(() => { controller.current?.abort(); setState({ data: null, loading: true, error: null }); setRefresh(value => value + 1); }, []);
  useEffect(() => {
    let disposed = false, stopped = false, timer: ReturnType<typeof setTimeout> | undefined;
    async function read() {
      const request = new AbortController(); controller.current = request;
      try {
        const data = await load(request.signal);
        if (!disposed && !request.signal.aborted) setState({ data, loading: false, error: null });
      } catch (error) {
        if (!disposed && !request.signal.aborted) {
          stopped = true; setState({ data: null, loading: false, error });
          if (error instanceof APIError && error.status === 401) void refreshUser(true).catch(() => {});
        }
        return; // Failed/unauthorized reads require explicit retry.
      }
      if (!disposed && !request.signal.aborted && poll) timer = setTimeout(() => { if (document.visibilityState === 'visible') void read(); }, 10000);
    }
    setState({ data: null, loading: true, error: null }); void read();
    const visible = () => { if (document.visibilityState === 'visible' && poll && !stopped) { clearTimeout(timer); controller.current?.abort(); void read(); } };
    document.addEventListener('visibilitychange', visible);
    return () => { disposed = true; clearTimeout(timer); controller.current?.abort(); document.removeEventListener('visibilitychange', visible); };
  }, [load, refresh, poll, refreshUser]);
  return { ...state, retry };
}
