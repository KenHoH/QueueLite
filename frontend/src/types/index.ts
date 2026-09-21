export type Service = "General" | "Haircut" | "Consultation";
export type QueueStatus =
  "Waiting" | "Serving" | "Completed" | "Skipped" | "Cancelled";
export interface Business {
  id: string;
  name: string;
  category: string;
  location: string;
  address: string;
  description: string;
  open: boolean;
  waiting: number;
  wait: string;
  counters: number;
  initials: string;
  color: string;
  phone: string;
  email: string;
}
export interface QueueEntry {
  id: string;
  businessId: string;
  name: string;
  priority: boolean;
  status: QueueStatus;
  service: Service;
  minutes: number;
  counter?: number;
  startedAt?: string;
  mine?: boolean;
}
export interface Counter {
  id: number;
  operator: string;
  started: string;
}
