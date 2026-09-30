import { APIError } from '../api/errors';
import { RegistrationSignInError } from '../state/authStore';
export interface AuthFields { username: string; phonenumber: string; email: string; password: string; confirmPassword: string }
export function validateAuth(fields: AuthFields, registering: boolean) {
  const errors: Partial<Record<keyof AuthFields, string>> = {};
  if (!fields.username.trim()) errors.username = 'Enter your username.';
  if (registering && !fields.phonenumber.trim()) errors.phonenumber = 'Enter your phone number.';
  if (registering && fields.email.trim() && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(fields.email.trim())) errors.email = 'Enter a valid email address.';
  if (!fields.password) errors.password = 'Enter your password.';
  if (registering && !fields.confirmPassword) errors.confirmPassword = 'Confirm your password.';
  else if (registering && fields.password !== fields.confirmPassword) errors.confirmPassword = 'Passwords must match.';
  return errors;
}
export function authErrorMessage(error: unknown, registering = false) {
  if (error instanceof RegistrationSignInError) return error.message;
  if (error instanceof APIError) {
    const messages: Record<string, string> = { INVALID_CREDENTIALS: 'Incorrect username or password.', USERNAME_REQUIRED: 'Enter your username.', PASSWORD_REQUIRED: 'Enter your password.', PHONENUMBER_REQUIRED: 'Enter your phone number.' };
    if (messages[error.code]) return messages[error.code];
  }
  return registering ? 'We could not complete registration. Please try again, or sign in if your account was already created.' : 'We could not complete sign-in. Please try again.';
}
