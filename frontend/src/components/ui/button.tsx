import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';

const buttonVariants = cva(
  'ql-button inline-flex min-h-11 items-center justify-center gap-2 rounded-[var(--ql-radius-control)] px-6 py-2 text-sm font-semibold no-underline transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ql-action)] focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-60',
  {
    variants: {
      variant: {
        default: 'bg-[var(--ql-action)] !text-white hover:opacity-90',
        secondary: 'border border-[var(--ql-border)] bg-[var(--ql-surface)] !text-[var(--ql-text)] hover:bg-[var(--ql-background)]',
        destructive: 'bg-[var(--ql-error-text)] !text-white hover:opacity-90',
        ghost: '!text-[var(--ql-text)] hover:bg-[var(--ql-background)]',
        link: 'min-h-0 px-0 !text-[var(--ql-action)] underline-offset-4 hover:underline',
      },
      size: {
        default: 'min-h-11 px-6',
        sm: 'min-h-9 px-3',
        icon: 'size-11 p-0',
      },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  },
);

export interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement>, VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(({ className, variant, size, asChild = false, ...props }, ref) => {
  const Comp = asChild ? Slot : 'button';
  return <Comp ref={ref} className={cn(buttonVariants({ variant, size }), className)} {...props} />;
});
Button.displayName = 'Button';

export { Button, buttonVariants };
