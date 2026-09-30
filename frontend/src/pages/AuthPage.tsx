import { useRef, useState, type FormEvent } from 'react';
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom';
import { Button, Card, InlineError, Input, PageContainer, PasswordInput } from '../components/foundation';
import { useAppState } from '../state/AppState';
import { RegistrationSignInError } from '../state/authStore';
import { authErrorMessage, validateAuth, type AuthFields } from './authForm';

export function AuthPage({ registering = false }: { registering?: boolean }) {
  const { auth, login, register } = useAppState();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const requestedPath = params.get('returnTo') ?? '/';
  const returnTo = requestedPath.startsWith('/') && !requestedPath.startsWith('//') && !requestedPath.includes('\\') && !/^\/(login|register)(\?|$)/.test(requestedPath) ? requestedPath : '/';
  const [fields, setFields] = useState<AuthFields>({ username: '', phonenumber: '', email: '', password: '', confirmPassword: '' });
  const [errors, setErrors] = useState<Partial<Record<keyof AuthFields, string>>>({});
  const [error, setError] = useState('');
  const [created, setCreated] = useState(false);
  const [busy, setBusy] = useState(false);
  const submitting = useRef(false);
  const errorRef = useRef<HTMLDivElement>(null);
  if (auth.status === 'authenticated') return <Navigate to={returnTo} replace />;
  const bind = (name: keyof AuthFields) => ({ name, id: `auth-${name}`, value: fields[name], disabled: busy, error: errors[name], onChange: (event: React.ChangeEvent<HTMLInputElement>) => setFields(previous => ({ ...previous, [name]: event.target.value })) });
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting.current || created) return;
    const validation = validateAuth(fields, registering);
    setErrors(validation); setError('');
    const first = Object.keys(validation)[0];
    if (first) { (event.currentTarget.elements.namedItem(first) as HTMLInputElement)?.focus(); return; }
    submitting.current = true; setBusy(true);
    try {
      const credentials = { username: fields.username.trim(), password: fields.password };
      if (registering) await register({ ...credentials, phonenumber: fields.phonenumber.trim(), ...(fields.email.trim() ? { email: fields.email.trim() } : {}) });
      else await login(credentials);
      setFields(previous => ({ ...previous, password: '', confirmPassword: '' }));
      navigate(returnTo, { replace: true });
    } catch (failure) {
      if (failure instanceof RegistrationSignInError) {
        setCreated(true);
        setFields(previous => ({ ...previous, password: '', confirmPassword: '' }));
      }
      setError(authErrorMessage(failure, registering));
      requestAnimationFrame(() => errorRef.current?.focus());
    } finally { submitting.current = false; setBusy(false); }
  }
  return <PageContainer className="ql-auth-page"><Card className="ql-auth-card">
    <p className="ql-auth-eyebrow">QueueLite · A little less waiting</p>
    <h1>{registering ? 'Create your QueueLite account.' : 'Welcome back.'}</h1>
    <p className="ql-muted">{registering ? 'Save your queues, manage your profile, and unlock more ways to wait less.' : 'Sign in to keep track of your queues and make waiting a little easier.'}</p>
    <form className="ql-stack" onSubmit={submit} noValidate aria-busy={busy}>
      <Input label="Username" autoComplete="username" required {...bind('username')} />
      {registering && <><Input label="Phone number" type="tel" autoComplete="tel" required {...bind('phonenumber')} /><Input label="Email (optional)" type="email" autoComplete="email" {...bind('email')} /></>}
      <PasswordInput label="Password" autoComplete={registering ? 'new-password' : 'current-password'} required {...bind('password')} />
      {registering && <PasswordInput label="Confirm password" autoComplete="new-password" required {...bind('confirmPassword')} />}
      {error && <div ref={errorRef} tabIndex={-1}><InlineError>{error}</InlineError>{created && <Link to="/login">Continue to sign in</Link>}</div>}
      <Button type="submit" loading={busy} disabled={created}>{busy ? (registering ? 'Creating account…' : 'Signing in…') : (registering ? 'Create account' : 'Sign in')}</Button>
    </form>
    <p className="ql-auth-switch">{registering ? 'Already have an account? ' : 'Don’t have an account? '}<Link to={registering ? '/login' : '/register'}>{registering ? 'Sign in' : 'Create one'}</Link></p>
    <div className="ql-auth-guest"><Link to="/">Continue as guest</Link></div>
  </Card></PageContainer>;
}
