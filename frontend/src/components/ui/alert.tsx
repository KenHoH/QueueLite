import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';

const alertVariants = cva('relative w-full rounded-[var(--ql-radius-control)] border p-4 text-sm', { variants: { variant: { default: 'border-[var(--ql-border)] bg-[var(--ql-surface)]', destructive: 'border-[var(--ql-error)] bg-red-50 text-[var(--ql-error-text)]' } }, defaultVariants: { variant: 'default' } });
const Alert = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement> & VariantProps<typeof alertVariants>>(({ className, variant, ...props }, ref) => <div ref={ref} role="alert" className={cn(alertVariants({ variant }), className)} {...props} />);
Alert.displayName = 'Alert';
const AlertTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(({ className, ...props }, ref) => <h5 ref={ref} className={cn('mb-1 font-semibold', className)} {...props} />);
AlertTitle.displayName = 'AlertTitle';
const AlertDescription = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => <div ref={ref} className={cn('text-sm', className)} {...props} />);
AlertDescription.displayName = 'AlertDescription';
export { Alert, AlertTitle, AlertDescription };
