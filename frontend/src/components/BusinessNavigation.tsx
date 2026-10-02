import { NavLink } from 'react-router-dom';
export function BusinessNavigation({ businessId, manager = true }: { businessId: string; manager?: boolean }) {
  const base = `/business/${businessId}`;
  return <nav className="ql-business-nav" aria-label="Business navigation"><NavLink to={`${base}/dashboard`}>Dashboard</NavLink><NavLink to={`${base}/counters`}>Counters</NavLink>{manager && <><NavLink to={`${base}/settings`}>Settings</NavLink><NavLink to={`${base}/plans`}>Plans</NavLink></>}</nav>;
}
