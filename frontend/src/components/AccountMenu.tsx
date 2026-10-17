import { useEffect, useId, useRef, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ChevronDown, LogOut } from 'lucide-react';
import { Button } from './foundation';

// Disclosure-style account menu: one trigger (avatar + name) that reveals account links.
// On phones the header already lives inside the hamburger panel, so the same list is shown
// inline there (see `.ql-account-menu` in foundation.css) instead of as a popover.
export function AccountMenu({ username, busy, onSignOut }: { username: string; busy: boolean; onSignOut: () => void }) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const panelId = useId();
  const location = useLocation();
  useEffect(() => setOpen(false), [location.pathname]);
  useEffect(() => {
    if (!open) return;
    const onPointer = (event: PointerEvent) => { if (!root.current?.contains(event.target as Node)) setOpen(false); };
    const onKey = (event: KeyboardEvent) => { if (event.key === 'Escape') { setOpen(false); trigger.current?.focus(); } };
    document.addEventListener('pointerdown', onPointer);
    document.addEventListener('keydown', onKey);
    return () => { document.removeEventListener('pointerdown', onPointer); document.removeEventListener('keydown', onKey); };
  }, [open]);
  return <div className="ql-account-menu" ref={root}>
    <button ref={trigger} type="button" className="ql-account-trigger" aria-expanded={open} aria-controls={panelId} aria-label={`Account menu for ${username}`} onClick={() => setOpen(value => !value)}>
      <span className="ql-avatar" aria-hidden="true">{username.slice(0, 1).toUpperCase()}</span>
      <span className="ql-username" aria-hidden="true">{username}</span>
      <ChevronDown className="ql-account-chevron" aria-hidden="true" />
    </button>
    <div id={panelId} className={`ql-account-panel ${open ? 'is-open' : ''}`}>
      <p className="ql-account-panel-name">Signed in as <strong>{username}</strong></p>
      <Link to="/profile">Account</Link>
      <Link to="/business/create">Create Business</Link>
      <Button variant="secondary" className="ql-account-signout" loading={busy} onClick={onSignOut}><LogOut aria-hidden="true" />Logout</Button>
    </div>
  </div>;
}
