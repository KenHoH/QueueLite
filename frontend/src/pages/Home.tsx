import { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Search, X } from 'lucide-react';
import { BusinessCard } from '../components/BusinessCard';
import { JoinedQueuesPreview } from '../components/JoinedQueuesPreview';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer, Spinner } from '../components/foundation';
import { Button as IconButton } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { useAppState } from '../state/AppState';
import { greeting } from './businessDisplay';
import { useBusinessDiscovery } from './useBusinessDiscovery';

export default function Home() {
  const { auth } = useAppState();
  const [query, setQuery] = useState('');
  const { hash } = useLocation();
  const discovery = useBusinessDiscovery(query);
  useEffect(() => { if (hash === '#businesses') document.getElementById('businesses')?.scrollIntoView(); }, [hash]);
  return <PageContainer className="ql-discovery-page">
    <section className="ql-discovery-intro">
      <p className="ql-eyebrow">QueueLite</p>
      <h1>{auth.status === 'authenticated' ? greeting(auth.user.username) : 'Find a business queue'}</h1>
      <p className="ql-muted">Search nearby businesses, join remotely, and keep your place from this device.</p>
    </section>
    <div className="ql-search">
      <Search size={21} aria-hidden="true" />
      <Label className="ql-sr-only" htmlFor="business-search">Search businesses</Label>
      <Input id="business-search" className="border-0 bg-transparent shadow-none focus-visible:ring-0" type="search" placeholder="Search businesses..." value={query} onChange={event => setQuery(event.target.value)} />
      {query && <IconButton variant="ghost" size="icon" type="button" aria-label="Clear search" onClick={() => setQuery('')}><X size={18} aria-hidden="true" /></IconButton>}
    </div>
    <JoinedQueuesPreview />
    <section id="businesses" className="ql-discovery-section" aria-labelledby="businesses-heading" aria-busy={discovery.loading || discovery.loadingMore}>
      <div className="ql-section-heading"><div><h2 id="businesses-heading">Discover businesses</h2><p className="ql-muted">Choose a business to see its details.</p></div>
        {discovery.loading && discovery.query && <Spinner label="Searching businesses" />}
      </div>
      {discovery.loading ? <div className="ql-business-grid">{Array.from({ length: 6 }, (_, index) => <Card key={index}><LoadingSkeleton label={discovery.query ? 'Loading search results' : 'Loading businesses'} lines={4} /></Card>)}</div>
        : discovery.error ? <Card><InlineError>We couldn’t load businesses right now.</InlineError><Button variant="secondary" onClick={discovery.retry}>Try again</Button></Card>
        : !discovery.businesses.length ? <EmptyState title={discovery.query ? `No businesses found for ‘${discovery.query}’.` : 'No businesses are available yet.'} description={discovery.query ? 'Try another search.' : 'Please check back later.'} />
        : <div className="ql-business-grid">{discovery.businesses.map(business => <BusinessCard key={business.id} business={business} />)}</div>}
      {!discovery.loading && discovery.cursor && !discovery.query && <div className="ql-load-more">
        {discovery.moreError && <InlineError>We couldn’t load more businesses. Your results are still here.</InlineError>}
        <Button variant="secondary" loading={discovery.loadingMore} onClick={() => { void discovery.loadMore(); }}>{discovery.moreError ? 'Try loading more again' : 'Load more'}</Button>
      </div>}
    </section>
  </PageContainer>;
}
