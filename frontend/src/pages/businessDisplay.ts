import type { Business } from '../api/types';

// Operational is backend availability, not a timezone-aware opening calculation.
export const businessStatus = (business: Business) => business.operational ? 'Operational' : 'Unavailable';
export const businessHours = (business: Business) => business.openTime && business.closeTime
  ? `${business.openTime} – ${business.closeTime}` : 'Hours not provided';
export function greeting(username: string, hour = new Date().getHours()) {
  return `Good ${hour < 12 ? 'morning' : hour < 18 ? 'afternoon' : 'evening'}, ${username}.`;
}
export const isBusinessId = (id: string | undefined): id is string =>
  !!id && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id);
