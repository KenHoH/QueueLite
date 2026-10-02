/** HTTP DTOs, not database records. Optional response fields are omitted, not null. */
export type UUID = string;
export type DateTime = string; // RFC3339 on responses
export type QueueState = 'waiting' | 'called' | 'processing' | 'cancelled' | 'skipped' | 'done';
/** Account membership returned by GET /businesses/mine. */
export type BusinessRole = 'owner' | 'admin' | 'counter';
export type SubscriptionStatus = 'active' | 'inactive';
export type SubscriptionType = 'business' | 'user';
export type BusinessPlanType = 'free' | 'plus' | 'pro' | 'max';
export type UserPlanType = 'standard' | 'premium';
export interface User { id: UUID; username: string; phonenumber: string; email?: string }
export interface Business {
  id: UUID; name: string; location: string; description?: string;
  operational: boolean; openTime: string; closeTime: string; email: string; phoneNumber: string;
}
export interface BusinessMembership extends Business { role: BusinessRole }
export interface BusinessMember { userId: UUID; username: string; role: BusinessRole }
export interface Queue {
  id: UUID; businessId: UUID; userId?: UUID; calledByCounterId?: UUID;
  name: string; state: QueueState; priority: boolean;
  calledAt?: DateTime; processingAt?: DateTime; doneAt?: DateTime; cancelledAt?: DateTime;
  createdAt?: DateTime; updatedAt?: DateTime;
}
export interface Counter {
  id: UUID; businessId: UUID; name: string; currentEmployeeId?: UUID; currentQueueId?: UUID;
}
export interface CustomerCounterStatus {
  id: UUID; name: string; currentQueueId?: UUID; currentQueueName?: string;
  state: 'idle' | 'called' | 'processing';
}
export interface CurrentlyServingQueue {
  counterId: UUID; counterName: string; queueId: UUID; queueName: string;
  state: 'called' | 'processing';
}
export interface CustomerQueueStatus {
  queue: Queue;
  business: { id: UUID; name: string; operational: boolean };
  counters: CustomerCounterStatus[];
  totalCounters: number;
  activeCounters: number;
  currentlyServing: CurrentlyServingQueue[];
  nextQueue?: { queueId: UUID; queueName: string };
  customerPosition?: { position: number; ahead: number; queueName: string };
  totalWaiting: number;
  totalActiveQueues: number;
  estimatedWaitMinutes?: number;
  updatedAt: DateTime;
}
export interface Subscription {
  id: UUID; businessId?: UUID; userId?: UUID; businessPlanId?: UUID; userPlanId?: UUID;
  type: SubscriptionType; startDate: DateTime; endDate?: DateTime; status: SubscriptionStatus;
}
export interface BusinessPlan {
  id: UUID; businessPlanType: BusinessPlanType; description: string; price: number;
  capacity: number; analysis: boolean; insight: boolean; prioritySupport: boolean;
  lastCapacityResetAt: DateTime;
}
export interface UserPlan {
  id: UUID; userPlanType: UserPlanType; name: string; description: string;
  price: number; slots: number; lastSlotsResetAt: DateTime;
}
export interface BusinessSubscriptionInfo { subscription: Subscription; businessPlan: BusinessPlan }
export interface UserSubscriptionInfo { subscription: Subscription; userPlan: UserPlan }
export interface QueueQRResolution {
  businessId: UUID; authenticated: boolean; requiresGuestForm: boolean;
  requiredFields?: Array<'username' | 'phoneNumber'>;
}
export interface Cursor { createdAt: DateTime; id: UUID }
export interface CursorPage<T> { data: T[]; nextCursor: Cursor | null }
export interface PageQuery { limit?: number; cursorCreatedAt?: DateTime; cursorID?: UUID }
export interface MessageResponse { message: string }
export type NextQueueResponse = { queue: null } | { queueId: UUID; queueName: string };
export interface ProcessQueueResponse { queueId: UUID; state: 'processing' }
export interface UseSubscriptionResponse { success: boolean; message: string }
export interface APIErrorResponse { error: { code: string; message: string } }

export interface LoginRequest { username: string; password: string; email?: string }
export interface RegisterUserRequest extends LoginRequest { phonenumber: string }
export interface UpdateUserRequest { phonenumber?: string; password?: string; email?: string }
export interface CreateBusinessRequest {
  name: string; location: string; description?: string | null; openTime: string;
  closeTime: string; email: string; phoneNumber: string;
}
export type UpdateBusinessRequest = Partial<Omit<CreateBusinessRequest, 'description'>> & { description?: string; operational?: boolean };
/** Legacy POST /queues/ delegates to customer joining. Browser identity is never supplied. */
export interface CreateQueueRequest { businessId: UUID; username?: string; phoneNumber?: string }
export interface GuestQueueRequest { username: string; phoneNumber: string }
/** Omit for account joining; provide both fields for a guest. */
export type CustomerQueueJoinRequest = GuestQueueRequest;
export interface UpdateQueueRequest { name?: string; state?: QueueState; priority?: boolean }
export interface CreateCounterRequest {
  businessId: UUID; name: string; currentEmployeeId?: UUID;
}
/** Empty string clears assignments; null is ignored by the handler. */
export interface UpdateCounterRequest { name?: string; currentEmployeeId?: UUID | '' }
export interface UpdateSubscriptionRequest { businessPlanId?: UUID; userPlanId?: UUID }
export interface UpdateSubscriptionTimeRequest { startDate: string; endDate?: string | null }
