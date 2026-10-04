import { cn } from '../../lib/utils';
export function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) { return <div className={cn('animate-pulse rounded-[var(--ql-radius-control)] bg-[var(--ql-border)]', className)} {...props} />; }
