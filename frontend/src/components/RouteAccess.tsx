import type { ReactNode } from 'react';
import { useParams } from 'react-router-dom';
import type { BusinessRole } from '../api/types';
import { useAppState } from '../state/AppState';
import { AccessScreen } from '../pages/AccessPage';
import { isBusinessId } from '../pages/businessDisplay';
import { Button } from './ui/button';

export function AccountRoute({ children, guestTitle, guestDescription }: { children: ReactNode; guestTitle?: string; guestDescription?: string }) {
  const { auth } = useAppState();
  if (auth.status === 'loading') return <AccessScreen kind="loading" title="Checking your account" />;
  if (auth.status === 'error') return <AccessScreen kind="session-error" />;
  if (auth.status === 'guest') return <AccessScreen kind="unauthorized" title={guestTitle} description={guestDescription} />;
  return children;
}

export function BusinessRoute({ children, roles = ['owner', 'admin', 'counter'] }: { children: ReactNode; roles?: BusinessRole[] }) {
  const { businessId } = useParams();
  const { businesses, businessesStatus, refreshBusinesses } = useAppState();
  return <AccountRoute>{!isBusinessId(businessId)
    ? <AccessScreen kind="not-found" title="Business not found" description="This business link is invalid." />
    : businessesStatus === 'loading'
      ? <AccessScreen kind="loading" title="Checking business access" />
      : businessesStatus === 'error'
        ? <AccessScreen kind="session-error" title="Business access unavailable" description="We couldn’t check your business membership." action={<Button type="button" variant="secondary" onClick={() => refreshBusinesses()}>Retry business access</Button>} />
        : !businesses.some(item => item.id === businessId && roles.includes(item.role))
          ? <AccessScreen kind="forbidden" description={roles.includes('counter') ? 'Business membership is required to view this page.' : 'Owner or admin access is required to manage this business.'} />
          : children}</AccountRoute>;
}
