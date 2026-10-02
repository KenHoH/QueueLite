import { APIError } from '../api/errors';
import type { GuestQueueRequest, QueueState } from '../api/types';

export const isActiveQueue = (state: QueueState) => ['waiting', 'called', 'processing'].includes(state);
export const queueHeadings: Record<QueueState, string> = {
  waiting: 'You’re in the queue.', called: 'It’s your turn.', processing: 'You’re being served.',
  done: 'Your visit is complete.', skipped: 'Your turn was skipped.', cancelled: 'Your queue is cancelled.',
};
export function validateGuestJoin(fields: GuestQueueRequest) {
  const errors: Partial<Record<keyof GuestQueueRequest, string>> = {};
  if (!fields.username.trim()) errors.username = 'Enter your name.';
  const phone = fields.phoneNumber.trim().replace(/[\s-]/g, '').replace(/^\+62/, '62').replace(/^08/, '628');
  if (!phone) errors.phoneNumber = 'Enter your phone number.';
  else if (!/^628[0-9]{8,11}$/.test(phone)) errors.phoneNumber = 'Enter an Indonesian mobile number, such as 0812 3456 7890.';
  return errors;
}
export function isAmbiguousJoin(error: unknown) {
  return !(error instanceof APIError) || error.status === 0 || error.status >= 500 || error.kind === 'response';
}
export function joinErrorMessage(error: unknown) {
  if (isAmbiguousJoin(error)) return 'We couldn’t confirm whether your place was saved. Please check your queue or ask the business before joining again.';
  const messages: Record<string, string> = {
    BUSINESS_NOT_FOUND: 'This business could not be found.', BUSINESS_UNAVAILABLE: 'This business is currently unavailable. Please try another time.',
    ACTIVE_QUEUE_EXISTS: 'You already have an active queue at this business.', BUSINESS_QUEUE_FULL: 'This business has reached its queue limit. Please try another time.',
    PHONE_ALREADY_REGISTERED: 'This phone number already has an active queue at this business.', INVALID_PHONE_NUMBER: 'Enter a valid Indonesian mobile number.',
    USERNAME_REQUIRED: 'Enter your name.', PHONE_NUMBER_REQUIRED: 'Enter your phone number.', INVALID_FORMAT: 'Please check your details and try again.',
  };
  return error instanceof APIError && messages[error.code] || 'We couldn’t join this queue. Please check your details and try again.';
}
export function ticketErrorMessage(error: unknown) {
  if (error instanceof APIError) {
    if (error.status === 401 || error.status === 403) return 'This ticket isn’t available in this session. Sign in to the account that joined, or open it in the browser you used as a guest.';
    if (error.status === 404 || error.code === 'INVALID_QUEUE_ID') return 'This queue ticket could not be found.';
  }
  return 'We couldn’t refresh your queue. Please try again.';
}
export function cancellationErrorMessage(error: unknown) {
  if (error instanceof APIError && error.code === 'QUEUE_NOT_CANCELLABLE') return 'This queue is no longer waiting. Refresh your ticket and ask the team for help.';
  if (error instanceof APIError && error.code === 'QUEUE_PERSISTENCE_PENDING') return 'Your ticket is still being saved. Please wait a moment, then try leaving again.';
  if (error instanceof APIError && [401, 403, 404].includes(error.status)) return ticketErrorMessage(error);
  return 'We couldn’t confirm that your queue was cancelled. Refresh your ticket before trying again.';
}
