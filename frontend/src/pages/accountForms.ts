import { APIError } from '../api/errors';
import type { Business, BusinessMembership } from '../api/types';

export const validEmail = (value: string) => /^[^<>\s@]+@[^<>\s@]+\.[^<>\s@]+$/.test(value.trim());
export const validPhone = (value: string) => /^\+?\d{8,15}$/.test(value.replace(/[\s().-]/g, ''));
export function validateProfile(fields: { phonenumber: string; email: string }) {
  const errors: Partial<Record<'phonenumber' | 'email', string>> = {};
  if (!validPhone(fields.phonenumber)) errors.phonenumber = 'Enter a phone number with 8–15 digits.';
  if (fields.email.trim() && !validEmail(fields.email)) errors.email = 'Enter a valid email address.';
  return errors;
}
export function validatePassword(password: string, confirmation: string) {
  const errors: Partial<Record<'password' | 'confirmation', string>> = {};
  if (!password.trim()) errors.password = 'Enter a new password.';
  else if (new TextEncoder().encode(password).length > 72) errors.password = 'Use a password of at most 72 bytes.';
  if (password !== confirmation || !confirmation) errors.confirmation = 'Passwords must match.';
  return errors;
}
export type BusinessFields = Pick<Business, 'name' | 'location' | 'openTime' | 'closeTime' | 'email' | 'phoneNumber' | 'operational'> & { description: string };
export const emptyBusiness: BusinessFields = { name: '', location: '', description: '', openTime: '', closeTime: '', email: '', phoneNumber: '', operational: true };
export function validateBusiness(fields: BusinessFields) {
  const errors: Partial<Record<keyof BusinessFields, string>> = {};
  if (!fields.name.trim()) errors.name = 'Enter a business name.';
  if (!fields.location.trim()) errors.location = 'Enter a location.';
  if (!validEmail(fields.email)) errors.email = 'Enter a valid email address.';
  if (!validPhone(fields.phoneNumber)) errors.phoneNumber = 'Enter a phone number with 8–15 digits.';
  for (const field of ['openTime', 'closeTime'] as const) if (!/^([01]\d|2[0-3]):[0-5]\d$/.test(fields[field])) errors[field] = 'Enter a valid time.';
  return errors;
}
export const canManage = (business: BusinessMembership) => business.role === 'owner' || business.role === 'admin';
export function accountError(error: unknown) {
  if (error instanceof APIError) {
    if (error.status === 401) return 'Your session has ended. Please sign in again.';
    if (error.status === 403) return 'You do not have permission to manage this business or account.';
    if (error.status === 404) return 'These details could not be found.';
    const messages: Record<string, string> = { BUSINESS_NAME_REQUIRED: 'Enter a business name.', BUSINESS_LOCATION_REQUIRED: 'Enter a location.', BUSINESS_EMAIL_REQUIRED: 'Enter an email address.', BUSINESS_PHONE_NUMBER_REQUIRED: 'Enter a phone number.', INVALID_OPEN_TIME: 'Enter a valid opening time.', INVALID_CLOSE_TIME: 'Enter a valid closing time.', NO_BUSINESS_FIELDS: 'Enter the details to update.' };
    if (messages[error.code]) return messages[error.code];
  }
  return 'We couldn’t complete this request. Please try again.';
}
