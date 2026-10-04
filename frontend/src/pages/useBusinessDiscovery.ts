import { useEffect, useRef, useState } from 'react';
import { getBusinesses, searchBusinesses } from '../api/businesses';
import type { Business, Cursor } from '../api/types';

const PAGE_LIMIT = 12;
interface DiscoveryState {
  query: string; businesses: Business[]; cursor: Cursor | null;
  loading: boolean; loadingMore: boolean; error: boolean; moreError: boolean;
}
const initial = (query: string): DiscoveryState => ({ query, businesses: [], cursor: null, loading: true, loadingMore: false, error: false, moreError: false });

export function useBusinessDiscovery(query: string) {
  const name = query.trim();
  const [state, setState] = useState(() => initial(name));
  const [attempt, setAttempt] = useState(0);
  const request = useRef<{ controller: AbortController; name: string; moreBusy: boolean } | null>(null);
  useEffect(() => {
    const current = { controller: new AbortController(), name, moreBusy: false };
    request.current = current;
    setState(initial(name));
    async function load() {
      try {
        const options = { signal: current.controller.signal };
        const page = name ? { data: await searchBusinesses(name, options), nextCursor: null }
          : await getBusinesses({ limit: PAGE_LIMIT }, options);
        if (!current.controller.signal.aborted) setState({ ...initial(name), businesses: page.data, cursor: page.nextCursor, loading: false });
      } catch {
        if (!current.controller.signal.aborted) setState({ ...initial(name), loading: false, error: true });
      }
    }
    // A query change aborts immediately, including while its replacement is debounced.
    const timer = window.setTimeout(() => { void load(); }, name ? 300 : 0);
    return () => { window.clearTimeout(timer); current.controller.abort(); };
  }, [name, attempt]);

  async function loadMore() {
    const current = request.current;
    if (!current || current.controller.signal.aborted || current.name !== name || name || !state.cursor || state.loading || current.moreBusy) return;
    current.moreBusy = true;
    setState(previous => ({ ...previous, loadingMore: true, moreError: false }));
    try {
      const page = await getBusinesses({ limit: PAGE_LIMIT, cursorCreatedAt: state.cursor.createdAt, cursorID: state.cursor.id }, { signal: current.controller.signal });
      if (!current.controller.signal.aborted) setState(previous => {
        const seen = new Set(previous.businesses.map(business => business.id));
        return { ...previous, businesses: [...previous.businesses, ...page.data.filter(business => !seen.has(business.id))], cursor: page.nextCursor, loadingMore: false };
      });
    } catch {
      if (!current.controller.signal.aborted) setState(previous => ({ ...previous, loadingMore: false, moreError: true }));
    } finally { current.moreBusy = false; }
  }
  // Hide prior-query content even in the render before effect cleanup runs.
  return { ...(state.query === name ? state : initial(name)), loadMore, retry: () => setAttempt(value => value + 1) };
}
