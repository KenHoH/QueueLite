import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cn } from '../../lib/utils';

type CardProps = React.HTMLAttributes<HTMLElement> & { asChild?: boolean };
function Card({ className, asChild = false, ...props }: CardProps) {
  const Comp = asChild ? Slot : 'div';
  return <Comp className={cn('rounded-[var(--ql-radius-card)] border border-[var(--ql-border)] bg-[var(--ql-surface)] text-[var(--ql-text)]', className)} {...props} />;
}
const CardHeader = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => <div ref={ref} className={cn('flex flex-col gap-1.5 p-6', className)} {...props} />);
CardHeader.displayName = 'CardHeader';
const CardTitle = React.forwardRef<HTMLHeadingElement, React.HTMLAttributes<HTMLHeadingElement>>(({ className, ...props }, ref) => <h3 ref={ref} className={cn('text-xl font-semibold leading-none', className)} {...props} />);
CardTitle.displayName = 'CardTitle';
const CardDescription = React.forwardRef<HTMLParagraphElement, React.HTMLAttributes<HTMLParagraphElement>>(({ className, ...props }, ref) => <p ref={ref} className={cn('text-sm text-[var(--ql-text-secondary)]', className)} {...props} />);
CardDescription.displayName = 'CardDescription';
const CardContent = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => <div ref={ref} className={cn('p-6 pt-0', className)} {...props} />);
CardContent.displayName = 'CardContent';
const CardFooter = React.forwardRef<HTMLDivElement, React.HTMLAttributes<HTMLDivElement>>(({ className, ...props }, ref) => <div ref={ref} className={cn('flex items-center p-6 pt-0', className)} {...props} />);
CardFooter.displayName = 'CardFooter';
export { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter };
