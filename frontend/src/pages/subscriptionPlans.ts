import type { BusinessMembership } from '../api/types';
import { canManage } from './accountForms';

export interface PlanComparison {
  plans: string[];
  features: Array<{ name: string; values: string[] }>;
}

export const userPlanComparison: PlanComparison = {
  plans: ['Standard', 'Premium'],
  features: [
    { name: 'Price', values: ['Free', '$4.99/month'] },
    { name: 'Join Normal Queue', values: ['✓', '✓'] },
    { name: 'Live Queue Tracking', values: ['✓', '✓'] },
    { name: 'WhatsApp Integration', values: ['✓', '✓'] },
    { name: 'Priority Passes', values: ['—', '4/month'] },
    { name: 'Priority Access', values: ['—', '✓'] },
    { name: 'Extra Priority Pass', values: ['$1.99 each', '$1.99 each'] },
  ],
};

export const businessPlanComparison: PlanComparison = {
  plans: ['Free', 'Plus', 'Pro', 'Max'],
  features: [
    { name: 'Price', values: ['Free', '$2.99/month', '$6.99/month', '$8.99/month'] },
    { name: 'Queue Capacity', values: ['100/day', '500/day', '2,000/day', 'Up to 10,000/day'] },
    { name: 'Web & QR Queue', values: ['✓', '✓', '✓', '✓'] },
    { name: 'WhatsApp Integration', values: ['—', '✓', '✓', '✓'] },
    { name: 'Queue Analytics', values: ['Basic', 'Basic', 'Advanced', 'Advanced'] },
    { name: 'AI Weekly Insights', values: ['—', '—', '—', '✓'] },
    { name: 'Priority Queue Support', values: ['Optional', '✓', '✓', '✓'] },
  ],
};

export const canShowBusinessPlans = (businesses: BusinessMembership[]) => businesses.some(canManage);
