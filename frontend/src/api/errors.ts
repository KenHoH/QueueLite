export type APIErrorKind = 'unauthorized' | 'forbidden' | 'not-found' | 'conflict' | 'validation' | 'server' | 'network' | 'response' | 'http';
const statusKinds: Partial<Record<number, APIErrorKind>> = { 400: 'validation', 401: 'unauthorized', 403: 'forbidden', 404: 'not-found', 409: 'conflict' };
/** Preserves backend codes, including future codes. Synthetic codes have a CLIENT_ prefix. */
export class APIError extends Error {
  readonly status: number;
  readonly code: string;
  readonly kind: APIErrorKind;
  constructor(status: number, code: string, message: string, kind?: APIErrorKind, cause?: unknown) {
    super(message, { cause });
    this.name = 'APIError';
    this.status = status;
    this.code = code;
    this.kind = kind ?? statusKinds[status] ?? (status >= 500 ? 'server' : 'http');
  }
}
const record = (value: unknown): value is Record<string, unknown> => typeof value === 'object' && value !== null;
export function parseAPIError(status: number, body: unknown): APIError {
  if (record(body) && record(body.error) && typeof body.error.code === 'string' && typeof body.error.message === 'string') {
    return new APIError(status, body.error.code, body.error.message);
  }
  // Middleware uses plain text; never show arbitrary proxy HTML to the user.
  const message = status === 401 ? 'Authentication is required.' : status === 403 ? 'Access is denied.' : `Request failed (${status}).`;
  return new APIError(status, `CLIENT_HTTP_${status}`, message);
}
export const queueErrorCodes = {
	 businessNotFound: ['BUSINESS_NOT_FOUND'],
	 businessUnavailable: ['BUSINESS_UNAVAILABLE'],
	 invalidPhone: ['INVALID_PHONE_NUMBER'],
	 usernameRequired: ['USERNAME_REQUIRED'],
	 phoneRequired: ['PHONE_NUMBER_REQUIRED', 'PHONENUMBER_REQUIRED'],
	 accessRequired: ['QUEUE_ACCESS_REQUIRED', 'USER_NOT_FOUND'],
	 accessDenied: ['QUEUE_ACCESS_DENIED'],
	 persistencePending: ['QUEUE_PERSISTENCE_PENDING'],
  full: ['BUSINESS_QUEUE_FULL', 'QUEUE_FULL'],
  activeQueueExists: ['ACTIVE_QUEUE_EXISTS'],
  noPrioritySlot: ['INVALID_PRIORITY_QUOTA', 'NO_PRIORITY_SLOT'],
  phoneReserved: ['PHONE_ALREADY_REGISTERED'],
} as const;
