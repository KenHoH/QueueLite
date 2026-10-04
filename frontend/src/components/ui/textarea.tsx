import * as React from 'react';
import { cn } from '../../lib/utils';

const Textarea = React.forwardRef<HTMLTextAreaElement, React.TextareaHTMLAttributes<HTMLTextAreaElement>>(({ className, ...props }, ref) => <textarea ref={ref} className={cn('flex min-h-24 w-full rounded-[var(--ql-radius-control)] border border-[var(--ql-border)] bg-[var(--ql-surface)] px-3 py-2 text-base text-[var(--ql-text)] outline-none placeholder:text-[var(--ql-text-secondary)] focus-visible:ring-2 focus-visible:ring-[var(--ql-action)] disabled:cursor-not-allowed disabled:opacity-60 aria-invalid:border-[var(--ql-error)]', className)} {...props} />);
Textarea.displayName = 'Textarea';
export { Textarea };
