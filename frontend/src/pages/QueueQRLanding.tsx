import { useEffect, useState } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { APIError } from '../api/errors';
import { resolveQueueQR } from '../api/queues';
import type { QueueQRResolution } from '../api/types';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer } from '../components/foundation';
import { isBusinessId } from './businessDisplay';

export default function QueueQRLanding() {
  const { businessId = '' } = useParams();
  const [resolution, setResolution] = useState<QueueQRResolution | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    if (!isBusinessId(businessId)) { setError(new APIError(400, 'INVALID_BUSINESS_ID', 'Invalid business link.')); return; }
    const controller = new AbortController();
    setError(null); setResolution(null);
    void resolveQueueQR(businessId, { signal: controller.signal }).then(value => {
      if (!controller.signal.aborted) setResolution(value);
    }).catch(failure => { if (!controller.signal.aborted) setError(failure); });
    return () => controller.abort();
  }, [businessId, attempt]);
  if (resolution) return <Navigate to={`/business/${resolution.businessId}/join`} replace />;
  return <PageContainer className="ql-ticket-page">
    {!error ? <Card><p className="ql-eyebrow">QUEUE QR</p><h1>Opening this business queue</h1><LoadingSkeleton label="Checking queue QR code" lines={3} /></Card>
      : error instanceof APIError && [400, 404].includes(error.status) ? <EmptyState title="Queue QR unavailable" description="This QR code is invalid or its business could not be found." action={<Link to="/#businesses">Browse businesses</Link>} />
      : <Card><h1>We couldn’t open this queue</h1><InlineError>Please check your connection and try scanning again.</InlineError><Button variant="secondary" onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>}
  </PageContainer>;
}
