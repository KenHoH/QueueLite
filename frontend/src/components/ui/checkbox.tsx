import * as React from 'react';
import * as CheckboxPrimitive from '@radix-ui/react-checkbox';
import { Check } from 'lucide-react';
import { cn } from '../../lib/utils';
const Checkbox = React.forwardRef<React.ElementRef<typeof CheckboxPrimitive.Root>, React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>>(({ className, ...props }, ref) => <CheckboxPrimitive.Root ref={ref} className={cn('peer size-5 shrink-0 rounded border border-[var(--ql-border)] bg-[var(--ql-surface)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--ql-action)] disabled:cursor-not-allowed disabled:opacity-60 data-[state=checked]:border-[var(--ql-action)] data-[state=checked]:bg-[var(--ql-action)] data-[state=checked]:text-white', className)} {...props}><CheckboxPrimitive.Indicator className="flex items-center justify-center"><Check className="size-4" /></CheckboxPrimitive.Indicator></CheckboxPrimitive.Root>);
Checkbox.displayName = CheckboxPrimitive.Root.displayName;
export { Checkbox };
