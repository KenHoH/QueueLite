import { useId, useState, type ButtonHTMLAttributes, type InputHTMLAttributes, type HTMLAttributes, type ReactNode } from 'react';
import { Eye, EyeOff, LoaderCircle } from 'lucide-react';
import type { QueueState } from '../api/types';
export function Spinner({ label = 'Loading' }: { label?: string }) {
  return <span className="ql-spinner" role="status"><LoaderCircle size={18} aria-hidden="true" /><span className="ql-sr-only">{label}</span></span>;
}
type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary' | 'danger'; loading?: boolean };
export function Button({ variant = 'primary', loading = false, disabled, type = 'button', className = '', children, ...props }: ButtonProps) {
  return <button {...props} type={type} className={`ql-button ql-button-${variant} ${className}`} disabled={disabled || loading} aria-busy={loading || undefined}>{loading && <Spinner />}{children}</button>;
}
export type InputProps = InputHTMLAttributes<HTMLInputElement> & { label: string; hint?: string; error?: string };
export function Input({ label, hint, error, id, className = '', ...props }: InputProps) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const describedBy = [props['aria-describedby'], hint && `${inputId}-hint`, error && `${inputId}-error`].filter(Boolean).join(' ') || undefined;
  return <div className="ql-field"><label htmlFor={inputId}>{label}</label>
    <input {...props} id={inputId} className={`ql-input ${className}`} aria-invalid={error ? true : props['aria-invalid']} aria-describedby={describedBy} />
    {hint && <span id={`${inputId}-hint`} className="ql-meta">{hint}</span>}
    {error && <span id={`${inputId}-error`} className="ql-error" role="alert">{error}</span>}
  </div>;
}
export function PasswordInput(props: Omit<InputProps, 'type'>) {
  const [visible, setVisible] = useState(false);
  const generatedId = useId();
  const inputId = props.id ?? generatedId;
  return <div className="ql-password"><Input autoComplete="current-password" {...props} id={inputId} type={visible ? 'text' : 'password'} />
    <button className="ql-password-toggle" type="button" disabled={props.disabled} aria-controls={inputId} aria-label={visible ? 'Hide password' : 'Show password'} aria-pressed={visible} onClick={() => setVisible(!visible)}>
      {visible ? <EyeOff size={18} aria-hidden="true" /> : <Eye size={18} aria-hidden="true" />}
    </button></div>;
}
export function Card({ className = '', ...props }: HTMLAttributes<HTMLDivElement>) { return <div {...props} className={`ql-card ${className}`} />; }
export function PageContainer({ className = '', ...props }: HTMLAttributes<HTMLElement>) { return <main {...props} className={`ql-page ${className}`} />; }
const queueLabels: Record<QueueState, string> = { waiting: 'Waiting', called: 'Called', processing: 'In service', cancelled: 'Cancelled', skipped: 'Skipped', done: 'Completed' };
export function StatusBadge({ state }: { state: QueueState }) { return <span className={`ql-badge ql-badge-${state}`}>{queueLabels[state]}</span>; }
export function LoadingSkeleton({ label = 'Loading content', lines = 3 }: { label?: string; lines?: number }) {
  return <div role="status" className="ql-stack"><span className="ql-sr-only">{label}</span>{Array.from({ length: Math.max(1, Math.min(10, lines)) }, (_, i) => <div key={i} className="ql-skeleton" aria-hidden="true" />)}</div>;
}
export function EmptyState({ title, description, action }: { title: string; description: string; action?: ReactNode }) { return <Card><h2>{title}</h2><p className="ql-muted">{description}</p>{action}</Card>; }
export function InlineError({ children }: { children: ReactNode }) { return <p className="ql-error" role="alert">{children}</p>; }
