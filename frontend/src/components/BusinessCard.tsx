import { ArrowRight, Clock3, MapPin, Store } from 'lucide-react';
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
export function BusinessCard({ business }: { business: Business }) {
  return <Card asChild className="ql-card ql-business-card"><Link to={`/business/${encodeURIComponent(business.id)}`}>
    <div className="ql-business-card-heading"><span className="ql-business-icon" aria-hidden="true"><Store size={25} /></span><BusinessAvailability business={business} /></div>
    <h3>{business.name}</h3>
    <p className="ql-business-location"><MapPin size={16} aria-hidden="true" />{business.location || 'Location not provided'}</p>
    {business.description && <p className="ql-business-description">{business.description}</p>}
    <div className="ql-business-card-footer"><span><Clock3 size={16} aria-hidden="true" />{businessHours(business)}</span>
      <span className="ql-view-business">View business <ArrowRight size={17} aria-hidden="true" /></span>
    </div>
  </Link></Card>;
}
