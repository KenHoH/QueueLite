import * as React from 'react';
import { cn } from '../../lib/utils';

const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(({ className, type, ...props }, ref) => <input type={type} ref={ref} className={cn('flex min-h-11 w-full rounded-[var(--ql-radius-control)] border border-[var(--ql-border)] bg-[var(--ql-surface)] px-3 py-2 text-base text-[var(--ql-text)] outline-none placeholder:text-[var(--ql-text-secondary)] focus-visible:ring-2 focus-visible:ring-[var(--ql-action)] disabled:cursor-not-allowed disabled:opacity-60 aria-invalid:border-[var(--ql-error)]', className)} {...props} />);
Input.displayName = 'Input';
export { Input };
