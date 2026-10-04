import { useEffect, useState } from 'react';
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom';
import { Menu, X } from 'lucide-react';
import { useAppState } from '../state/AppState';
import { Button, InlineError, Spinner } from './foundation';
import { Button as IconButton } from './ui/button';

export function CustomerHeader() {
  const { auth, logout, refreshUser, businesses } = useAppState();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [menuOpen, setMenuOpen] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  useEffect(() => setMenuOpen(false), [location.pathname]);
  async function signOut() {
    setBusy(true); setError('');
    try { await logout(); navigate('/'); }
    catch { setError('We could not sign you out. Please try again.'); }
    finally { setBusy(false); }
  }
  return <>
    <a className="ql-skip-link" href="#main-content">Skip to main content</a>
    <header className="ql-header"><div className="ql-header-row">
      <Link className="ql-brand" to="/" aria-label="QueueLite home">Queue<span>Lite</span></Link>
      <IconButton className="ql-menu-toggle" variant="ghost" size="icon" type="button" aria-label={menuOpen ? 'Close navigation' : 'Open navigation'} aria-expanded={menuOpen} aria-controls="site-navigation" onClick={() => setMenuOpen(value => !value)}>{menuOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}</IconButton>
      <div id="site-navigation" className={`ql-header-navigation ${menuOpen ? 'is-open' : ''}`}>
        <nav className="ql-customer-nav" aria-label="Primary navigation"><NavLink to="/" end>Home</NavLink>{auth.status === 'authenticated' ? <NavLink to="/my-queues">My queues</NavLink> : <Link to="/#businesses">Businesses</Link>}</nav>
        <nav className="ql-account" aria-label="Account navigation">
          {auth.status === 'loading' && <Spinner label="Checking your session" />}
          {auth.status === 'guest' && <><Link to="/login">Sign in</Link><Button asChild><Link to="/register">Create account</Link></Button></>}
          {auth.status === 'authenticated' && <><span className="ql-user-chip"><span className="ql-avatar" aria-hidden="true">{auth.user.username.slice(0, 1).toUpperCase()}</span><span className="ql-username">{auth.user.username}</span></span><Link to="/profile">Account</Link>{businesses.length > 0 && <><Link to="/counters">My Counters</Link><Link to="/business/manage">Manage Business</Link></>}<Link to="/business/create">Create Business</Link><Button variant="secondary" loading={busy} onClick={signOut}>Logout</Button></>}
          {auth.status === 'error' && <Button variant="secondary" onClick={() => { void refreshUser().catch(() => {}); }}>Retry session check</Button>}
        </nav>
      </div>
    </div>{auth.status === 'error' && <InlineError>We could not check your session. You can still browse QueueLite. Please retry.</InlineError>}{error && <InlineError>{error}</InlineError>}</header>
  </>;
}
