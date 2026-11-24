import { useEffect, useState } from 'react';
import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom';
import { Menu, Volume2, VolumeX, X } from 'lucide-react';
import { motion } from 'motion/react';
import { useAppState } from '../state/AppState';
import { useSound } from '../state/SoundContext';
import { Button, InlineError, Spinner } from './foundation';
import { useToast } from './Toast';
import { Button as IconButton } from './ui/button';
import { AccountMenu } from './AccountMenu';

function AnimatedNavLink({ to, end = false, children }: { to: string; end?: boolean; children: string }) {
  const sound = useSound();
  const { pathname } = useLocation();
  return (
    <NavLink to={to} end={end} onClick={() => sound.tap()}>
      {({ isActive }) => (
        <>
          <span>{children}</span>
          {isActive && (
            <motion.span
              layoutId="header-nav-indicator"
              layoutDependency={pathname}
              className="ql-nav-indicator"
              aria-hidden="true"
              transition={{ type: 'spring', stiffness: 420, damping: 34 }}
            />
          )}
        </>
      )}
    </NavLink>
  );
}

export function CustomerHeader() {
  const { auth, logout, refreshUser, businesses } = useAppState();
  const sound = useSound();
  const toast = useToast();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [menuOpen, setMenuOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  useEffect(() => setMenuOpen(false), [location.pathname]);
  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 12);
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);
  async function signOut() {
    setBusy(true); setError('');
    try { await logout(); navigate('/'); }
    catch { setError('We could not sign you out. Please try again.'); }
    finally { setBusy(false); }
  }
  function handleSoundToggle() {
    const next = sound.toggle('pop');
    toast.info(next ? 'Sound effects on.' : 'Sound effects muted.');
  }
  return <>
    <a className="ql-skip-link" href="#main-content">Skip to main content</a>
    <header className={`ql-header ${scrolled ? 'is-scrolled' : ''}`}><div className="ql-header-row">
      <Link className="ql-brand" to="/" aria-label="QueueLite home">Queue<span>Lite</span></Link>
      <IconButton className="ql-menu-toggle" variant="ghost" size="icon" type="button" aria-label={menuOpen ? 'Close navigation' : 'Open navigation'} aria-expanded={menuOpen} aria-controls="site-navigation" onClick={() => { sound.tap(); setMenuOpen(value => !value); }}><span key={menuOpen ? 'close' : 'open'} className="ql-icon-pop">{menuOpen ? <X aria-hidden="true" /> : <Menu aria-hidden="true" />}</span></IconButton>
      <div id="site-navigation" className={`ql-header-navigation ${menuOpen ? 'is-open' : ''}`}>
        <div className="ql-header-navigation-inner">
          <nav className="ql-customer-nav" aria-label="Primary navigation">
            <AnimatedNavLink to="/" end>Home</AnimatedNavLink>
            {auth.status === 'authenticated' ? <>
              <AnimatedNavLink to="/my-queues">My queues</AnimatedNavLink>
              {businesses.length > 0 && <>
                <AnimatedNavLink to="/counters">My Counters</AnimatedNavLink>
                <AnimatedNavLink to="/business/manage">Manage Business</AnimatedNavLink>
              </>}
            </> : <Link to="/#businesses" onClick={() => sound.tap()}>Businesses</Link>}
          </nav>
          <nav className="ql-account" aria-label="Account navigation">
            <button
              type="button"
              className={`ql-header-sound ${sound.enabled ? 'is-on' : ''}`}
              aria-label={sound.enabled ? 'Mute sound effects' : 'Enable sound effects'}
              aria-pressed={sound.enabled}
              title={sound.enabled ? 'Sound effects on' : 'Sound effects off'}
              onClick={handleSoundToggle}
            >
              <span key={sound.enabled ? 'on' : 'off'} className="ql-icon-pop">
                {sound.enabled ? <Volume2 size={17} aria-hidden="true" /> : <VolumeX size={17} aria-hidden="true" />}
              </span>
              <span className="ql-header-sound-label">{sound.enabled ? 'Sound on' : 'Sound off'}</span>
            </button>
            {auth.status === 'loading' && <Spinner label="Checking your session" />}
            {auth.status === 'guest' && <><Link to="/login">Sign in</Link><Button asChild><Link to="/register">Create account</Link></Button></>}
            {auth.status === 'authenticated' && <AccountMenu username={auth.user.username} busy={busy} onSignOut={signOut} />}
            {auth.status === 'error' && <Button variant="secondary" onClick={() => { void refreshUser().catch(() => {}); }}>Retry session check</Button>}
          </nav>
        </div>
      </div>
    </div>{auth.status === 'error' && <InlineError>We could not check your session. You can still browse QueueLite. Please retry.</InlineError>}{error && <InlineError>{error}</InlineError>}</header>
  </>;
}
