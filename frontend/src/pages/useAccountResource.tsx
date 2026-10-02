import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { APIError } from '../api/errors';
import { Button, Card, InlineError, LoadingSkeleton } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { accountError } from './accountForms';

export function useAccountResource<T>(load: (signal: AbortSignal) => Promise<T>) {
  const { refreshUser } = useAppState();
  const [state, setState] = useState<{ data: T | null; loading: boolean; error: unknown }>({ data: null, loading: true, error: null });
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setState({ data: null, loading: true, error: null });
    void load(controller.signal).then(data => {
      if (!controller.signal.aborted) setState({ data, loading: false, error: null });
    }).catch(error => {
      if (controller.signal.aborted) return;
      setState({ data: null, loading: false, error });
      if (error instanceof APIError && error.status === 401) void refreshUser(true).catch(() => {});
    });
    return () => controller.abort();
  }, [load, attempt, refreshUser]);
  return { ...state, retry: () => setAttempt(value => value + 1) };
}
export function ResourceFeedback({ loading, error, retry }: { loading: boolean; error: unknown; retry: () => void }) {
  if (loading) return <LoadingSkeleton label="Loading account details" />;
  if (!error) return null;
  return <Card><InlineError>{accountError(error)}</InlineError>{error instanceof APIError && error.status === 401 ? <Link to="/login">Sign in</Link> : <Button variant="secondary" onClick={retry}>Try again</Button>}</Card>;
}
