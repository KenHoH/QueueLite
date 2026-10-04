import { APIError } from './errors';

type RecordValue = Record<string, unknown>;
type Schema = (value: unknown) => boolean;
const record = (value: unknown): value is RecordValue => !!value && typeof value === 'object' && !Array.isArray(value);
const text = (value: unknown) => typeof value === 'string';
const nonempty = (value: unknown) => text(value) && (value as string).trim().length > 0;
const optional = (value: RecordValue, keys: string[], check: Schema = text) => keys.every(key => value[key] === undefined || check(value[key]));
const strings = (value: RecordValue, keys: string[]) => keys.every(key => text(value[key]));
const role = (value: unknown) => ['owner', 'admin', 'counter'].includes(value as string);
const state = (value: unknown) => ['waiting', 'called', 'processing', 'done', 'cancelled', 'skipped'].includes(value as string);
const number = (value: unknown) => typeof value === 'number' && Number.isFinite(value) && value >= 0;
const user: Schema = value => record(value) && nonempty(value.id) && nonempty(value.username) && nonempty(value.phonenumber) && optional(value, ['email']);
const business: Schema = value => record(value) && nonempty(value.id) && strings(value, ['name', 'location']) && typeof value.operational === 'boolean' && optional(value, ['description', 'email', 'phoneNumber', 'openTime', 'closeTime']);
const membership: Schema = value => business(value) && role((value as RecordValue).role);
const queue: Schema = value => record(value) && nonempty(value.id) && nonempty(value.businessId) && nonempty(value.name) && state(value.state) && typeof value.priority === 'boolean' && optional(value, ['userId', 'calledByCounterId', 'calledAt', 'processingAt', 'doneAt', 'cancelledAt', 'createdAt', 'updatedAt']);
const counter: Schema = value => record(value) && nonempty(value.id) && nonempty(value.businessId) && nonempty(value.name) && optional(value, ['currentQueueId', 'currentEmployeeId']);
const customerCounter: Schema = value => record(value) && nonempty(value.id) && text(value.name) && ['idle', 'called', 'processing'].includes(value.state as string) && optional(value, ['currentQueueId', 'currentQueueName']);
const customerStatus: Schema = value => record(value) && queue(value.queue) && record(value.business) && nonempty(value.business.id) && text(value.business.name) && typeof value.business.operational === 'boolean'
  && array(customerCounter)(value.counters) && number(value.totalCounters) && number(value.activeCounters) && Array.isArray(value.currentlyServing)
  && number(value.totalWaiting) && number(value.totalActiveQueues) && text(value.updatedAt)
  && optional(value, ['estimatedWaitMinutes'], number);
const member: Schema = value => record(value) && nonempty(value.userId) && text(value.username) && role(value.role);
const subscription: Schema = value => record(value) && ['user', 'business'].includes(value.type as string) && ['active', 'inactive'].includes(value.status as string) && text(value.startDate) && optional(value, ['id', 'endDate', 'userId', 'businessId']);
const planInfo = (kind: 'user' | 'business'): Schema => value => {
  if (!record(value) || !subscription(value.subscription)) return false;
  const plan = value[`${kind}Plan`];
  return record(plan) && (kind === 'user' ? ['standard', 'premium'].includes(plan.userPlanType as string) && number(plan.slots) : ['free', 'plus', 'pro', 'max'].includes(plan.businessPlanType as string) && number(plan.capacity))
    && optional(plan, ['id', 'name', 'description', 'lastSlotsResetAt', 'lastCapacityResetAt']) && optional(plan, ['price'], number) && optional(plan, ['analysis', 'insight', 'prioritySupport'], v => typeof v === 'boolean');
};
const array = (schema: Schema): Schema => value => Array.isArray(value) && value.every(schema);
const cursor: Schema = value => value === null || (record(value) && strings(value, ['id', 'createdAt']));
const page = (schema: Schema): Schema => value => record(value) && array(schema)(value.data) && cursor(value.nextCursor);
const message: Schema = value => record(value) && text(value.message);
const next: Schema = value => record(value) && (value.queue === null || (nonempty(value.queueId) && text(value.queueName)));
const unique = (values: RecordValue[]) => [...new Map(values.map(value => [value.id ?? value.userId, value])).values()];

/** Validate consumed wire fields before React or mutation success can trust them. */
export function validateAPIResponse(path: string, method: string, value: unknown, status = 200): unknown {
  let schema: Schema | undefined;
  if (/^\/users\/(me|[^/]+)$/.test(path) && method === 'GET' || path === '/users/' && method === 'POST') schema = user;
  else if (path === '/businesses/mine') schema = array(membership);
  else if (path === '/businesses/search') schema = array(business);
  else if (path === '/businesses/' && method === 'GET') schema = page(business);
  else if (/^\/businesses\/[^/]+\/counters$/.test(path)) schema = array(counter);
  else if (/^\/businesses\/[^/]+\/members$/.test(path) && method === 'GET') schema = array(member);
  else if (/^\/businesses\/[^/]+\/members$/.test(path) && method === 'PUT') schema = member;
  else if (/^\/businesses\/[^/]+\/queues$/.test(path)) schema = array(queue);
  else if (/^\/businesses\/[^/]+$/.test(path) && method === 'GET' || path === '/businesses/' && method === 'POST') schema = business;
  else if (path === '/queues/me' || /^\/queues\/business\/[^/]+$/.test(path)) schema = array(queue);
  else if (/^\/queues\/[^/]+\/state$/.test(path) && method === 'GET') schema = v => record(v) && state(v.state);
  else if (/^\/queues\/[^/]+\/customer-status$/.test(path) && method === 'GET') schema = customerStatus;
  else if (/^\/queues\/qr\/[^/]+\/resolve$/.test(path)) schema = v => record(v) && nonempty(v.businessId) && typeof v.authenticated === 'boolean' && typeof v.requiresGuestForm === 'boolean';
  else if (/^\/queues\/[^/]+$/.test(path) && method === 'GET' || method === 'POST' && (path === '/queues/' || /^\/queues\/(qr\/[^/]+|business\/[^/]+\/join)$/.test(path))) schema = queue;
  else if (/^\/counters\/[^/]+$/.test(path) && method === 'GET' || path === '/counters/' && method === 'POST') schema = counter;
  else if (/^\/counters\/.+\/(call-next|skip)$/.test(path)) schema = next;
  else if (/^\/counters\/.+\/process$/.test(path)) schema = v => record(v) && nonempty(v.queueId) && v.state === 'processing';
  else if (/^\/subscriptions\/users\/[^/]+$/.test(path)) schema = planInfo('user');
  else if (/^\/subscriptions\/businesses\/[^/]+$/.test(path)) schema = planInfo('business');
  else if (method !== 'GET' && !path.endsWith('/use')) schema = message;
  if (schema && !schema(value)) throw new APIError(status, 'CLIENT_INVALID_RESPONSE', 'QueueLite returned an unexpected response.', 'response');
  if (schema && Array.isArray(value)) return unique(value);
  if (schema && record(value) && Array.isArray(value.data)) return { ...value, data: unique(value.data) };
  return value;
}
