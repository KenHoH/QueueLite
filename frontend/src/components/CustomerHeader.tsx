import { useState } from 'react';
import { Link, NavLink, useNavigate } from 'react-router-dom';
import { useAppState } from '../state/AppState';
import { Button, InlineError, Spinner } from './foundation';
export function CustomerHeader() {
  const { auth, logout, refreshUser, businesses } = useAppState();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();
  async function signOut() {
    setBusy(true); setError('');
    try { await logout(); navigate('/'); }
    catch { setError('We could not sign you out. Please try again.'); }
    finally { setBusy(false); }
  }
  return <header className="ql-header"><div className="ql-header-row">
    <Link className="ql-brand" to="/">QueueLite</Link>
    <nav className="ql-customer-nav" aria-label="Customer navigation"><NavLink to="/" end>Home</NavLink>{auth.status === 'authenticated' ? <NavLink to="/my-queues">My Queues</NavLink> : <Link to="/#businesses">Businesses</Link>}</nav>
    <nav className="ql-account" aria-label="Account">
      {auth.status === 'loading' && <Spinner label="Checking your session" />}
      {auth.status === 'guest' && <><Link to="/login">Sign in</Link><Link className="ql-button ql-button-primary" to="/register">Create account</Link></>}
      {auth.status === 'authenticated' && <><span className="ql-avatar" aria-hidden="true">{auth.user.username.slice(0, 1).toUpperCase()}</span><span className="ql-username">{auth.user.username}</span><Link to="/profile">Profile</Link>{businesses.length > 0 && <Link to="/business/manage">Manage Business</Link>}<Button variant="secondary" loading={busy} onClick={signOut}>Logout</Button></>}
      {auth.status === 'error' && <Button variant="secondary" onClick={() => { void refreshUser().catch(() => {}); }}>Retry session check</Button>}
    </nav>
  </div>{auth.status === 'error' && <InlineError>We could not check your session. You can still browse QueueLite. Please retry.</InlineError>}{error && <InlineError>{error}</InlineError>}</header>;
}
