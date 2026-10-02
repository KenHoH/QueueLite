import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, ArrowRight, Clock3, Mail, MapPin, Phone } from 'lucide-react';
import { getBusiness } from '../api/businesses';
import { APIError } from '../api/errors';
import type { Business } from '../api/types';
import { BusinessAvailability } from '../components/BusinessCard';
import { BusinessQueueSummary } from '../components/BusinessQueueSummary';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer } from '../components/foundation';
import { businessHours, isBusinessId } from './businessDisplay';

type DetailState = { [Status in 'loading' | 'not-found' | 'error']: { id: string | undefined; status: Status } }['loading' | 'not-found' | 'error'] | { id: string; status: 'ready'; business: Business };
export default function BusinessDetail() {
  const { businessId } = useParams();
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<DetailState>({ id: businessId, status: 'loading' });
  useEffect(() => {
    if (!isBusinessId(businessId)) { setState({ id: businessId, status: 'not-found' }); return; }
    const controller = new AbortController();
    setState({ id: businessId, status: 'loading' });
    void getBusiness(businessId, { signal: controller.signal }).then(business => {
      if (!controller.signal.aborted) setState({ id: businessId, status: 'ready', business });
    }).catch(error => {
      if (!controller.signal.aborted) setState({ id: businessId, status: error instanceof APIError && (error.status === 404 || error.code === 'BUSINESS_NOT_FOUND' || error.code === 'INVALID_BUSINESS_ID') ? 'not-found' : 'error' });
    });
    return () => controller.abort();
  }, [businessId, attempt]);
  const current = state.id === businessId ? state : { status: 'loading' as const };
  const back = <Link className="ql-back-link" to="/#businesses"><ArrowLeft size={16} aria-hidden="true" />Back to businesses</Link>;
  return <PageContainer className="ql-business-detail">{back}
    {current.status === 'loading' ? <Card><LoadingSkeleton label="Loading business details" lines={7} /></Card>
      : current.status === 'not-found' ? <EmptyState title="Business not found." description="This business may no longer be available." action={<Link to="/#businesses">Back to businesses</Link>} />
      : current.status === 'error' ? <Card><InlineError>We couldn’t load this business.</InlineError><Button variant="secondary" onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>
      : <div className="ql-detail-layout"><section className="ql-detail-content">
        <BusinessAvailability business={current.business} />
        <h1>{current.business.name}</h1>
        <p className="ql-business-location"><MapPin size={18} aria-hidden="true" />{current.business.location || 'Location not provided'}</p>
        <Card className="ql-detail-description"><h2>About this business</h2><p className="ql-muted">{current.business.description || 'No description provided.'}</p></Card>
      </section><aside className="ql-card ql-business-information"><h2>Business information</h2>
        <dl><div><dt><Clock3 size={18} aria-hidden="true" />Hours</dt><dd>{businessHours(current.business)}</dd></div>
          <div><dt><Phone size={18} aria-hidden="true" />Phone</dt><dd>{current.business.phoneNumber ? <a href={`tel:${current.business.phoneNumber.replace(/[^+\d]/g, '')}`}>{current.business.phoneNumber}</a> : 'Not provided'}</dd></div>
          <div><dt><Mail size={18} aria-hidden="true" />Email</dt><dd>{current.business.email ? <a href={`mailto:${current.business.email}`}>{current.business.email}</a> : 'Not provided'}</dd></div></dl>
        <BusinessQueueSummary businessId={current.business.id} />
        {current.business.operational ? <Link className="ql-button ql-button-primary ql-join-link" to={`/business/${encodeURIComponent(current.business.id)}/join`}>Join queue <ArrowRight size={17} aria-hidden="true" /></Link> : <Button className="ql-join-link" disabled>Join queue</Button>}
        <p className="ql-meta ql-join-note">{current.business.operational ? 'Save your place and keep your ticket handy.' : 'This business is currently unavailable.'}</p>
      </aside></div>}
  </PageContainer>;
}
