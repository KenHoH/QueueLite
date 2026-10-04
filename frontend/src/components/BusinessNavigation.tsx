import { ExternalLink } from 'lucide-react';
import { NavLink } from 'react-router-dom';

export function BusinessNavigation({ businessId, manager = true }: { businessId: string; manager?: boolean }) {
  const base = `/business/${businessId}`;
  return <nav className="ql-business-nav" aria-label="Business tools">
    <NavLink to={`${base}/dashboard`}>Overview</NavLink>
    <NavLink to={`${base}/counters`}>Counters</NavLink>
    {manager && <><NavLink to={`${base}/members`}>Members</NavLink><NavLink to={`${base}/settings`}>Settings</NavLink><NavLink to={`${base}/plans`}>Subscription</NavLink><NavLink to={`${base}/share`}>Share queue</NavLink></>}
    <NavLink to={base}>Public page <ExternalLink size={14} aria-hidden="true" /></NavLink>
  </nav>;
}
