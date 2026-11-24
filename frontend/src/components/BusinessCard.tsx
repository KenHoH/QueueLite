import { ArrowRight, Clock3, MapPin, Store } from 'lucide-react';
import type { CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import type { Business } from '../api/types';
import { businessHours, businessStatus } from '../pages/businessDisplay';
import { Badge } from './ui/badge';
import { Card } from './ui/card';

export function BusinessAvailability({ business }: { business: Business }) {
  return <Badge variant={business.operational ? 'default' : 'secondary'} className={`ql-availability ${business.operational ? '' : 'ql-unavailable'}`}>
    <span className="ql-status-dot" aria-hidden="true" />{businessStatus(business)}
  </Badge>;
}
export function BusinessCard({ business, index = 0 }: { business: Business; index?: number }) {
  const staggerStyle = { '--ql-stagger': `${Math.min(index, 6) * 35}ms` } as CSSProperties;
  return <Card asChild className="ql-card ql-business-card ql-card-stagger" style={staggerStyle}><Link viewTransition to={`/business/${encodeURIComponent(business.id)}`}>
    <div className="ql-business-card-heading"><span className="ql-business-icon" style={{ viewTransitionName: `ql-biz-icon-${business.id}` }} aria-hidden="true"><Store size={25} /></span><BusinessAvailability business={business} /></div>
    <h3 style={{ viewTransitionName: `ql-biz-title-${business.id}` }}>{business.name}</h3>
    <p className="ql-business-location"><MapPin size={16} aria-hidden="true" />{business.location || 'Location not provided'}</p>
    {business.description && <p className="ql-business-description">{business.description}</p>}
    <div className="ql-business-card-footer"><span><Clock3 size={16} aria-hidden="true" />{businessHours(business)}</span>
      <span className="ql-view-business">View business <ArrowRight size={17} aria-hidden="true" /></span>
    </div>
  </Link></Card>;
}
