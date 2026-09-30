import { ArrowUpRight, Clock3, MapPin } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { Business } from '../api/types';
import { businessHours, businessStatus } from '../pages/businessDisplay';

export function BusinessAvailability({ business }: { business: Business }) {
  return <span className={`ql-badge ql-availability ${business.operational ? '' : 'ql-unavailable'}`}>
    <span className="ql-status-dot" aria-hidden="true" />{businessStatus(business)}
  </span>;
}
export function BusinessCard({ business }: { business: Business }) {
  return <Link className="ql-card ql-business-card" to={`/business/${encodeURIComponent(business.id)}`}>
    <BusinessAvailability business={business} />
    <h3>{business.name}</h3>
    <p className="ql-business-location"><MapPin size={16} aria-hidden="true" />{business.location || 'Location not provided'}</p>
    {business.description && <p className="ql-business-description">{business.description}</p>}
    <div className="ql-business-card-footer"><span><Clock3 size={16} aria-hidden="true" />{businessHours(business)}</span>
      <span className="ql-view-business">View business <ArrowUpRight size={17} aria-hidden="true" /></span>
    </div>
  </Link>;
}
