import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from 'react';
import { ArrowLeft, ArrowUpRight, ChevronDown, MapPin } from 'lucide-react';
import { Link, useLocation } from 'react-router-dom';
import { useAppState } from '../state/AppState';
import { BusinessNavigation } from './BusinessNavigation';
import { CustomerHeader } from './CustomerHeader';

const WorkspaceBusiness = createContext<(id: string | undefined) => void>(() => {});
/** The counter read already knows its business; reuse it instead of another API request. */
export function useWorkspaceBusiness(id: string | undefined) {
  const setBusiness = useContext(WorkspaceBusiness);
  useEffect(() => { if (id) setBusiness(id); }, [id, setBusiness]);
}
export function ApplicationShell({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  const { businesses } = useAppState();
  const match = pathname.match(/^\/business\/([^/]+)\/(dashboard|counters|members|settings|plans|share)\/?$/);
  const isCounter = /^\/counter\/[^/]+\/?$/.test(pathname);
  const workspace = !!match || isCounter;
  const [counterBusiness, setCounterBusiness] = useState<{ path: string; id?: string }>();
  const setBusiness = useCallback((id: string | undefined) => setCounterBusiness({ path: pathname, id }), [pathname]);
  const businessId = match?.[1] ?? (counterBusiness?.path === pathname ? counterBusiness.id : undefined);
  const business = businesses.find(item => item.id === businessId);
  const [open, setOpen] = useState(false);
  useEffect(() => setOpen(false), [pathname]);
  if (!workspace) return <div className="ql-app"><CustomerHeader /><div className="ql-app-content">{children}</div><footer className="ql-footer"><span>QueueLite <span aria-hidden="true">·</span> Don’t wait in line. Join the line.</span><span><MapPin size={14} aria-hidden="true" /> Made for your neighborhood</span></footer></div>;
  return <WorkspaceBusiness.Provider value={setBusiness}><div className="ql-app ql-workspace-shell">
    <aside className={`ql-sidebar ${open ? 'is-open' : ''}`} aria-label="Business workspace">
      <div className="ql-sidebar-top"><Link className="ql-brand" to="/">Queue<span>Lite</span></Link><button className="ql-sidebar-toggle" type="button" aria-expanded={open} aria-controls="workspace-navigation" onClick={() => setOpen(value => !value)}>Workspace menu <ChevronDown className="ql-sidebar-chevron" size={16} aria-hidden="true" /></button></div>
      <div id="workspace-navigation" className="ql-sidebar-content">
        <div className="ql-sidebar-content-inner">
          <p className="ql-eyebrow">Business workspace</p>
          <Link className="ql-business-identity" to="/business/manage"><span className="ql-business-monogram" aria-hidden="true">{business?.name.charAt(0).toUpperCase() ?? 'Q'}</span><span><strong>{business?.name ?? 'Your workspace'}</strong><small>{business?.location ?? 'Choose a business'}</small></span></Link>
          {business ? <BusinessNavigation businessId={business.id} manager={business.role === 'owner' || business.role === 'admin'} /> : <nav className="ql-business-nav" aria-label="Business tools"><Link to="/business/manage">Your businesses <ArrowUpRight size={16} aria-hidden="true" /></Link><Link to="/counters">My counters</Link></nav>}
          <div className="ql-sidebar-bottom"><Link to="/business/manage">Switch business <ArrowUpRight size={14} aria-hidden="true" /></Link><Link to="/"><ArrowLeft size={16} aria-hidden="true" /> Customer view</Link></div>
        </div>
      </div>
    </aside>
    <div className="ql-workspace-body"><div className="ql-workspace-topbar"><span className="ql-breadcrumb">{business?.name ?? 'QueueLite'} <span aria-hidden="true">/</span> Workspace</span><CustomerHeader /></div><div className="ql-app-content">{children}</div></div>
  </div></WorkspaceBusiness.Provider>;
}
