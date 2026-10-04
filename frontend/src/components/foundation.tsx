import { useId, useState, type ButtonHTMLAttributes, type InputHTMLAttributes, type HTMLAttributes, type ReactNode } from 'react';
import { Eye, EyeOff, LoaderCircle } from 'lucide-react';
import type { QueueState } from '../api/types';
import { Alert, AlertDescription } from './ui/alert';
import { Badge } from './ui/badge';
import { Button as ShadcnButton, type ButtonProps as ShadcnButtonProps } from './ui/button';
import { Card as ShadcnCard } from './ui/card';
import { Input as ShadcnInput } from './ui/input';
import { Label } from './ui/label';
import { Skeleton } from './ui/skeleton';

export function Spinner({ label = 'Loading' }: { label?: string }) {
  return <span className="ql-spinner" role="status"><LoaderCircle className="size-[18px]" aria-hidden="true" /><span className="ql-sr-only">{label}</span></span>;
}

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'danger'; loading?: boolean; asChild?: boolean };
const buttonVariant: Record<NonNullable<ButtonProps['variant']>, ShadcnButtonProps['variant']> = { primary: 'default', secondary: 'secondary', danger: 'destructive' };
export function Button({ variant = 'primary', loading = false, disabled, type = 'button', asChild = false, className, children, ...props }: ButtonProps) {
  if (asChild) {
    return <ShadcnButton {...props} asChild variant={buttonVariant[variant]} className={className} disabled={disabled || loading} aria-busy={loading || undefined}>{children}</ShadcnButton>;
  }
  return <ShadcnButton {...props} type={type} variant={buttonVariant[variant]} className={className} disabled={disabled || loading} aria-busy={loading || undefined}>{loading && <Spinner />}{children}</ShadcnButton>;
}

export type InputProps = InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string; error?: string };
export function Input({ label, hint, error, id, className, ...props }: InputProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const describedBy = [props['aria-describedby'], hint && `${inputId}-hint`, error && `${inputId}-error`].filter(Boolean).join(' ') || undefined;
  return <div className="ql-field"><Label htmlFor={inputId}>{label}</Label>
    <ShadcnInput {...props} id={inputId} className={className} aria-invalid={error ? true : props['aria-invalid']} aria-describedby={describedBy} />
    {hint && <span id={`${inputId}-hint`} className="ql-meta">{hint}</span>}
    {error && <span id={`${inputId}-error`} className="ql-error" role="alert">{error}</span>}
  </div>;
}

export function PasswordInput(props: Omit<InputProps, 'type'>) {
  const [visible, setVisible] = useState(false);
  const generatedId = useId();
  const inputId = props.id ?? generatedId;
  return <div className="ql-password"><Input autoComplete="current-password" {...props} className={`pr-12 ${props.className ?? ''}`} id={inputId} type={visible ? 'text' : 'password'} />
    <ShadcnButton variant="ghost" size="icon" className="ql-password-toggle" type="button" disabled={props.disabled} aria-controls={inputId} aria-label={visible ? 'Hide password' : 'Show password'} aria-pressed={visible} onClick={() => setVisible(!visible)}>
      {visible ? <EyeOff className="size-[18px]" aria-hidden="true" /> : <Eye className="size-[18px]" aria-hidden="true" />}
    </ShadcnButton></div>;
}

export function Card({ className = '', asChild = false, ...props }: HTMLAttributes<HTMLElement> & { asChild?: boolean }) { return <ShadcnCard {...props} asChild={asChild} className={`ql-card ${className}`} />; }
export function PageContainer({ className = '', id = 'main-content', ...props }: HTMLAttributes<HTMLElement>) { return <main {...props} id={id} tabIndex={-1} className={`ql-page ${className}`} />; }
export function PageHeader({ title, description, eyebrow, action }: { title: string; description?: string; eyebrow?: string; action?: ReactNode }) {
  return <div className="ql-page-heading"><div>{eyebrow && <p className="ql-eyebrow">{eyebrow}</p>}<h1>{title}</h1>{description && <p className="ql-muted">{description}</p>}</div>{action && <div className="ql-page-action">{action}</div>}</div>;
}
const queueLabels: Record<QueueState, string> = { waiting: 'Waiting', called: 'Called', processing: 'In service', cancelled: 'Cancelled', skipped: 'Skipped', done: 'Completed' };
export function StatusBadge({ state }: { state: QueueState }) { return <Badge variant={state === 'cancelled' || state === 'skipped' ? 'secondary' : 'default'} className={`ql-badge-${state}`}>{queueLabels[state]}</Badge>; }
export function LoadingSkeleton({ label = 'Loading content', lines = 3 }: { label?: string; lines?: number }) {
  return <div role="status" className="ql-stack"><span className="ql-sr-only">{label}</span>{Array.from({ length: Math.max(1, Math.min(10, lines)) }, (_, i) => <Skeleton key={i} className="ql-skeleton h-6" aria-hidden="true" />)}</div>;
}
export function EmptyState({ title, description, action }: { title: string; description: string; action?: ReactNode }) { return <Card><h2>{title}</h2><p className="ql-muted">{description}</p>{action}</Card>; }
export function InlineError({ children }: { children: ReactNode }) { return <Alert variant="destructive" className="ql-error"><AlertDescription>{children}</AlertDescription></Alert>; }
