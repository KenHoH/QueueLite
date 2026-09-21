import {
  ArrowUpRight,
  LayoutDashboard,
  Monitor,
  ArrowLeft,
  MapPin,
} from "lucide-react";
import { Link, NavLink, Outlet, useLocation } from "react-router-dom";
export function Brand() {
  return (
    <Link to="/" className="brand" aria-label="QueueLite home">
      <img src="/queuelite-logo.png" alt="QueueLite" />
    </Link>
  );
}
export function CustomerLayout() {
  const location = useLocation();
  return (
    <>
      <header className="topbar">
        <div className="nav-inner">
          <Brand />
          <nav aria-label="Customer navigation">
            <NavLink to="/" end className={!location.hash ? undefined : ""}>
              Home
            </NavLink>
            <Link
              className={location.hash === "#my-queues" ? "active" : ""}
              to="/#my-queues"
            >
              My Queues
            </Link>
            <Link
              className={location.hash === "#businesses" ? "active" : ""}
              to="/#businesses"
            >
              Businesses
            </Link>
          </nav>
          <div className="nav-right">
            <Link to="/dashboard" className="demo-link">
              Business demo <ArrowUpRight size={15} />
            </Link>
            <span className="nav-divider" />
            <span className="avatar" title="Matthew Sutiono · demo profile">
              MS
            </span>
          </div>
        </div>
      </header>
      <main className="customer-main">
        <Outlet />
      </main>
      <footer className="footer">
        <span>
          QueueLite <span className="footer-dot">·</span> Don't wait in line.
          Join the line.
        </span>
        <span className="flex items-center gap-2">
          <MapPin size={13} /> Made for your neighborhood
        </span>
      </footer>
    </>
  );
}
export function BusinessLayout() {
  return (
    <div className="business-shell">
      <aside className="sidebar">
        <Brand />
        <span className="business-label">BUSINESS WORKSPACE</span>
        <div className="business-switch">
          <span className="mini-logo">N</span>
          <div>
            <strong>Northside Barbers</strong>
            <span>Pluit, Jakarta</span>
          </div>
        </div>
        <nav aria-label="Business navigation">
          <NavLink to="/dashboard">
            <LayoutDashboard size={18} />
            Overview
          </NavLink>
          <NavLink to="/counter/2">
            <Monitor size={18} />
            Counter workspace
          </NavLink>
        </nav>
        <div className="sidebar-bottom">
          <div className="prototype-note">
            <span className="status-dot" /> Prototype workspace
            <p>
              Local demo data.
              <br />A fresh start on every refresh.
            </p>
          </div>
          <Link to="/" className="customer-back">
            <ArrowLeft size={16} /> Customer view
          </Link>
        </div>
      </aside>
      <div className="business-body">
        <header className="business-topbar">
          <span>
            Northside Barbers <span className="slash">/</span> Workspace
          </span>
          <div className="flex items-center gap-4">
            <span className="badge badge-green">
              <span className="status-dot" />
              Business open
            </span>
            <span className="avatar">MS</span>
          </div>
        </header>
        <main className="dashboard-main">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
