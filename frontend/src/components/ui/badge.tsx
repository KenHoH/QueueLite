import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';
const badgeVariants = cva('ql-badge inline-flex items-center rounded-full px-3 py-1 text-xs font-semibold', { variants: { variant: { default: 'bg-[var(--ql-soft-mint)] text-[var(--ql-action)]', secondary: 'bg-[var(--ql-background)] text-[var(--ql-text)]', destructive: 'bg-red-50 text-[var(--ql-error-text)]', outline: 'border border-[var(--ql-border)] text-[var(--ql-text)]' } }, defaultVariants: { variant: 'default' } });
export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement>, VariantProps<typeof badgeVariants> {}
function Badge({ className, variant, ...props }: BadgeProps) { return <span className={cn(badgeVariants({ variant }), className)} {...props} />; }
export { Badge, badgeVariants };
