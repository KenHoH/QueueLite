import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer } from '../components/foundation';
import { useAppState } from '../state/AppState';

export type AccessKind = 'loading' | 'session-error' | 'unauthorized' | 'forbidden' | 'not-found';

export function AccessScreen({ kind, title, description, action }: { kind: AccessKind; title?: string; description?: string; action?: ReactNode }) {
  const location = useLocation();
  const { refreshUser } = useAppState();
  if (kind === 'loading') return <PageContainer><Card><LoadingSkeleton label={title ?? 'Checking access'} /></Card></PageContainer>;
  if (kind === 'session-error') return <PageContainer><Card><h1>{title ?? 'Session check unavailable'}</h1><InlineError>{description ?? 'We couldn’t confirm your session. Please try again.'}</InlineError>{action ?? <Button variant="secondary" onClick={() => { void refreshUser().catch(() => {}); }}>Retry session check</Button>}</Card></PageContainer>;
  const defaults = kind === 'unauthorized'
    ? { title: 'Sign in to continue.', description: 'Use your QueueLite account to open this page.' }
    : kind === 'forbidden'
      ? { title: 'Permission required', description: 'Your account does not have access to this page.' }
      : { title: 'Page not found', description: 'This page or resource could not be found.' };
  const defaultAction = kind === 'unauthorized'
    ? <Button asChild><Link to={`/login?returnTo=${encodeURIComponent(location.pathname + location.search)}`}>Sign in</Link></Button>
    : <Button asChild variant="secondary"><Link to={kind === 'forbidden' ? '/business/manage' : '/'}>{kind === 'forbidden' ? 'Your businesses' : 'Return home'}</Link></Button>;
  return <PageContainer><EmptyState title={title ?? defaults.title} description={description ?? defaults.description} action={action ?? defaultAction} /></PageContainer>;
}

export function UnauthorizedPage() { return <AccessScreen kind="unauthorized" />; }
export function ForbiddenPage() { return <AccessScreen kind="forbidden" />; }
