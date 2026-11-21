import { CreditCard, ExternalLink, LayoutGrid, Monitor, QrCode, Settings, Users } from 'lucide-react';
import { NavLink } from 'react-router-dom';

export function BusinessNavigation({ businessId, manager = true }: { businessId: string; manager?: boolean }) {
  const base = `/business/${businessId}`;
  return <nav className="ql-business-nav" aria-label="Business tools">
    <NavLink to={`${base}/dashboard`}><LayoutGrid size={17} aria-hidden="true" /> Overview</NavLink>
    <NavLink to={`${base}/counters`}><Monitor size={17} aria-hidden="true" /> Counters</NavLink>
    {manager && <><NavLink to={`${base}/members`}><Users size={17} aria-hidden="true" /> Members</NavLink><NavLink to={`${base}/settings`}><Settings size={17} aria-hidden="true" /> Settings</NavLink><NavLink to={`${base}/plans`}><CreditCard size={17} aria-hidden="true" /> Subscription</NavLink><NavLink to={`${base}/share`}><QrCode size={17} aria-hidden="true" /> Share queue</NavLink></>}
    <NavLink to={base} end><ExternalLink size={17} aria-hidden="true" /> Public page</NavLink>
  </nav>;
}
