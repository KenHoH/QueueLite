import { useRef, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { updateUser } from '../api/users';
import { APIError } from '../api/errors';
import type { User } from '../api/types';
import { AccountAccess } from '../components/AccountAccess';
import { Button, Card, InlineError, Input, PageContainer, PasswordInput } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { accountError, canManage, validatePassword, validateProfile } from './accountForms';
import { UserSubscriptionSummary } from './Plans';

export default function Profile() {
  const { auth } = useAppState();
  return <PageContainer className="ql-account-page"><h1>Profile</h1><AccountAccess>{auth.status === 'authenticated' && <ProfileContent key={auth.user.id} user={auth.user} />}</AccountAccess></PageContainer>;
}
function ProfileContent({ user }: { user: User }) {
  const { refreshUser, businesses, businessesStatus, refreshBusinesses } = useAppState();
  const [fields, setFields] = useState({ phonenumber: user.phonenumber, email: user.email ?? '' });
  const [errors, setErrors] = useState<ReturnType<typeof validateProfile>>({});
  const [password, setPassword] = useState('');
  const [confirmation, setConfirmation] = useState('');
  const [passwordErrors, setPasswordErrors] = useState<ReturnType<typeof validatePassword>>({});
  const [busy, setBusy] = useState<'contact' | 'password' | null>(null);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const lock = useRef(false);
  async function submit(event: FormEvent<HTMLFormElement>, kind: 'contact' | 'password') {
    event.preventDefault();
    if (lock.current) return;
    const contactValidation = validateProfile(fields);
    const passwordValidation = validatePassword(password, confirmation);
    const validation = kind === 'contact' ? contactValidation : passwordValidation;
    if (kind === 'contact') setErrors(contactValidation); else setPasswordErrors(passwordValidation);
    setError(''); setMessage('');
    const first = Object.keys(validation)[0];
    if (first) { (event.currentTarget.elements.namedItem(first) as HTMLInputElement)?.focus(); return; }
    lock.current = true; setBusy(kind);
    try {
      await updateUser(user.id, kind === 'contact' ? { phonenumber: fields.phonenumber.trim(), email: fields.email.trim() } : { password });
      if (kind === 'password') { setPassword(''); setConfirmation(''); }
      setMessage(kind === 'contact' ? 'Profile saved.' : 'Password changed.');
      if (kind === 'contact') await refreshUser(true);
    } catch (failure) {
      setError(accountError(failure));
      if (failure instanceof APIError && failure.status === 401) void refreshUser().catch(() => {});
    } finally { lock.current = false; setBusy(null); }
  }
  return <div className="ql-stack">
    <Card><h2>Your details</h2><form className="ql-stack" noValidate onSubmit={event => { void submit(event, 'contact'); }} aria-busy={busy === 'contact'}>
      <Input label="Username" value={user.username} readOnly hint="Your username is used to sign in." />
      <Input label="Phone number" name="phonenumber" type="tel" autoComplete="tel" value={fields.phonenumber} disabled={!!busy} required error={errors.phonenumber} onChange={event => setFields({ ...fields, phonenumber: event.target.value })} />
      <Input label="Email (optional)" name="email" type="email" autoComplete="email" value={fields.email} disabled={!!busy} error={errors.email} onChange={event => setFields({ ...fields, email: event.target.value })} />
      <Button type="submit" disabled={!!busy} loading={busy === 'contact'}>Save profile</Button>
    </form></Card>
    <Card><h2>Change password</h2><form className="ql-stack" noValidate onSubmit={event => { void submit(event, 'password'); }} aria-busy={busy === 'password'}>
      <PasswordInput label="New password" name="password" autoComplete="new-password" value={password} disabled={!!busy} required error={passwordErrors.password} onChange={event => setPassword(event.target.value)} />
      <PasswordInput label="Confirm new password" name="confirmation" autoComplete="new-password" value={confirmation} disabled={!!busy} required error={passwordErrors.confirmation} onChange={event => setConfirmation(event.target.value)} />
      <Button type="submit" disabled={!!busy} loading={busy === 'password'}>Change password</Button>
    </form></Card>
    {error && <InlineError>{error}</InlineError>}{message && <p role="status">{message}</p>}
    <UserSubscriptionSummary userId={user.id} /><Link to="/plans">Plans &amp; Subscription →</Link>
    <Card><h2>Your businesses</h2>{businessesStatus === 'error' ? <><InlineError>We couldn’t load your businesses.</InlineError><Button onClick={() => refreshBusinesses()}>Try again</Button></> : businessesStatus === 'loading' ? <p role="status">Loading your businesses…</p> : businesses.some(canManage) ? <p><Link to="/business/manage">Manage business →</Link></p> : <p className="ql-muted">{businesses.length ? 'Your staff access will be available in a later stage.' : 'You don’t own or manage a business yet.'}</p>}<Link to="/business/create">Create a business</Link></Card>
  </div>;
}
