import { useEffect, useRef, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Search, X } from 'lucide-react';
import { motion } from 'motion/react';
import { BusinessCard } from '../components/BusinessCard';
import { JoinedQueuesPreview } from '../components/JoinedQueuesPreview';
import { LiveClock } from '../components/LiveVisuals';
import { Button, Card, EmptyState, InlineError, LoadingSkeleton, PageContainer, Spinner } from '../components/foundation';
import { Button as IconButton } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Label } from '../components/ui/label';
import { useAppState } from '../state/AppState';
import { useSound } from '../state/SoundContext';
import { greeting } from './businessDisplay';
import { useBusinessDiscovery } from './useBusinessDiscovery';

type AvailabilityFilter = 'all' | 'operational';

export default function Home() {
  const { auth } = useAppState();
  const sound = useSound();
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState<AvailabilityFilter>('all');
  const [searchFocused, setSearchFocused] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);
  const { hash } = useLocation();
  const discovery = useBusinessDiscovery(query);

  useEffect(() => { if (hash === '#businesses') document.getElementById('businesses')?.scrollIntoView(); }, [hash]);

  // Press "/" anywhere outside form controls to jump straight to business search.
  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== '/' || event.ctrlKey || event.metaKey || event.altKey) return;
      const target = event.target as HTMLElement | null;
      if (target && (['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) || target.isContentEditable)) return;
      event.preventDefault();
      searchRef.current?.focus();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, []);

  const visibleBusinesses = filter === 'operational'
    ? discovery.businesses.filter(business => business.operational)
    : discovery.businesses;
  const operationalCount = discovery.businesses.filter(business => business.operational).length;

  function chooseFilter(next: AvailabilityFilter) {
    if (next === filter) return;
    sound.tap();
    setFilter(next);
  }

  return <PageContainer className="ql-discovery-page">
    <section className="ql-discovery-intro ql-hero-beat-1"><div>
      <p className="ql-eyebrow">More life. Less waiting.</p>
      <h1>{auth.status === 'authenticated' ? greeting(auth.user.username) : 'Find a business queue'}</h1>
      <p className="ql-muted">Find a business and join the queue without waiting in line.</p>
    </div><div className="ql-time-note"><LiveClock size={24} /><span>Your time.<br /><strong>Back in your hands.</strong></span></div></section>
    <div className="ql-search ql-hero-beat-2">
      <Search size={21} aria-hidden="true" />
      <Label className="ql-sr-only" htmlFor="business-search">Search businesses</Label>
      <Input
        ref={searchRef}
        id="business-search"
        className="border-0 bg-transparent shadow-none focus-visible:ring-0"
        type="search"
        placeholder="Search businesses..."
        value={query}
        onFocus={() => setSearchFocused(true)}
        onBlur={() => setSearchFocused(false)}
        onKeyDown={event => {
          if (event.key === 'Escape') {
            if (query) setQuery('');
            searchRef.current?.blur();
          }
        }}
        onChange={event => setQuery(event.target.value)}
      />
      {!query && !searchFocused && <kbd className="ql-kbd ql-search-kbd" aria-hidden="true">/</kbd>}
      {query && <IconButton className="ql-icon-pop" variant="ghost" size="icon" type="button" aria-label="Clear search" onClick={() => { sound.tap(); setQuery(''); }}><X size={18} aria-hidden="true" /></IconButton>}
    </div>
    <JoinedQueuesPreview />
    <section id="businesses" className="ql-discovery-section ql-hero-beat-3" aria-labelledby="businesses-heading" aria-busy={discovery.loading || discovery.loadingMore}>
      <div className="ql-section-heading">
        <div><h2 id="businesses-heading">Discover businesses</h2><p className="ql-muted">Local businesses. A better way to wait.</p></div>
        <div className="ql-discovery-tools">
          {discovery.loading && discovery.query && <Spinner label="Searching businesses" />}
          {!discovery.loading && discovery.businesses.length > 0 && (
            <div className="ql-filter-pills" role="group" aria-label="Filter businesses by availability">
              <button type="button" className={`ql-filter-pill ${filter === 'all' ? 'is-active' : ''}`} aria-pressed={filter === 'all'} onClick={() => chooseFilter('all')}>
                {filter === 'all' && <motion.span layoutId="home-filter-pill" layoutDependency={filter} className="ql-filter-pill-bg" aria-hidden="true" transition={{ type: 'spring', stiffness: 420, damping: 32 }} />}
                <span className="ql-filter-pill-label">All <span className="ql-filter-count">{discovery.businesses.length}</span></span>
              </button>
              <button type="button" className={`ql-filter-pill ${filter === 'operational' ? 'is-active' : ''}`} aria-pressed={filter === 'operational'} onClick={() => chooseFilter('operational')}>
                {filter === 'operational' && <motion.span layoutId="home-filter-pill" layoutDependency={filter} className="ql-filter-pill-bg" aria-hidden="true" transition={{ type: 'spring', stiffness: 420, damping: 32 }} />}
                <span className="ql-filter-pill-label">Operational <span className="ql-filter-count">{operationalCount}</span></span>
              </button>
            </div>
          )}
        </div>
      </div>
      {discovery.loading ? <div className="ql-business-grid">{Array.from({ length: 6 }, (_, index) => <Card key={index}><LoadingSkeleton label={discovery.query ? 'Loading search results' : 'Loading businesses'} lines={4} /></Card>)}</div>
        : discovery.error ? <Card><InlineError>We couldn’t load businesses right now.</InlineError><Button variant="secondary" onClick={discovery.retry}>Try again</Button></Card>
        : !discovery.businesses.length ? <EmptyState title={discovery.query ? `No businesses found for ‘${discovery.query}’.` : 'No businesses are available yet.'} description={discovery.query ? 'Try another search.' : 'Please check back later.'} />
        : !visibleBusinesses.length ? <EmptyState title="No operational businesses in this view." description="Switch back to all businesses or check again shortly." action={<Button variant="secondary" onClick={() => setFilter('all')}>Show all businesses</Button>} />
        : <div className="ql-business-grid">{visibleBusinesses.map((business, index) => <BusinessCard key={business.id} business={business} index={index} />)}</div>}
      {!discovery.loading && discovery.cursor && !discovery.query && <div className="ql-load-more">
        {discovery.moreError && <InlineError>We couldn’t load more businesses. Your results are still here.</InlineError>}
        <Button variant="secondary" loading={discovery.loadingMore} onClick={() => { void discovery.loadMore(); }}>{discovery.moreError ? 'Try loading more again' : 'Load more'}</Button>
      </div>}
    </section>
  </PageContainer>;
}
