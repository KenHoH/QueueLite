import { useCallback } from 'react';
import type { BusinessMembership } from '../api/types';
import { getBusinessMembers } from '../api/operations';
import { BusinessOperationsPage, OperationsFeedback } from './BusinessOperations';
import { MemberManagement } from './CounterManagement';
import { useOperations } from './useOperations';

export default function BusinessMembers() {
  return <BusinessOperationsPage title="Business members">{business => <Members business={business} />}</BusinessOperationsPage>;
}

function Members({ business }: { business: BusinessMembership }) {
  const load = useCallback((signal: AbortSignal) => getBusinessMembers(business.id, { signal }), [business.id]);
  const resource = useOperations(load, false);
  return <div className="ql-narrow-section">
    <p className="ql-muted">Control who can manage this business or operate an assigned counter.</p>
    <OperationsFeedback {...resource} />
    {resource.data && <MemberManagement business={business} members={resource.data} ready refresh={resource.retry} />}
  </div>;
}
