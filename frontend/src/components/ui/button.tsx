import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '../../lib/utils';

const buttonVariants = cva(
  'ql-button inline-flex min-h-11 items-center justify-center gap-2 rounded-[var(--ql-radius-control)] px-6 py-2 text-[var(--ql-small)] font-semibold no-underline transition-all duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--ql-action)] focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-60',
  {
    variants: {
      variant: {
        default: 'bg-[var(--ql-action)] !text-white shadow-[0_2px_8px_rgb(28_74_59_/_18%)] hover:bg-[var(--ql-action-hover)] hover:shadow-[0_6px_16px_-4px_rgb(28_74_59_/_28%)]',
        secondary: 'border border-[var(--ql-border)] bg-[var(--ql-surface)] !text-[var(--ql-text)] hover:border-[var(--ql-mint-border)] hover:bg-[var(--ql-soft-mint)] hover:!text-[var(--ql-action)]',
        destructive: 'bg-[var(--ql-error-text)] !text-white shadow-[0_2px_8px_rgb(155_44_44_/_18%)] hover:opacity-92',
        ghost: '!text-[var(--ql-text)] hover:bg-[var(--ql-soft-mint)] hover:!text-[var(--ql-action)]',
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
