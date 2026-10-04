import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { useAppState } from '../state/AppState';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton } from './foundation';

export function AccountAccess({ children }: { children: ReactNode }) {
  const { auth } = useAppState();
  const location = useLocation();
  if (auth.status === 'loading') return <Card><LoadingSkeleton label="Checking your account" /></Card>;
  if (auth.status === 'error') return <Card><InlineError>Please retry the session check above to continue.</InlineError></Card>;
  if (auth.status === 'guest') return <EmptyState title="Sign in to continue." description="Use your QueueLite account to manage your profile, plans, and businesses." action={<Button asChild><Link to={`/login?returnTo=${encodeURIComponent(location.pathname + location.search)}`}>Sign in</Link></Button>} />;
  return children;
}
