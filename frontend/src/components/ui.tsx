import {
  ArrowRight,
  Clock3,
  MapPin,
  Scissors,
  HeartPulse,
  Wrench,
  Sparkles,
} from "lucide-react";
import { Link } from "react-router-dom";
import type { Business, QueueEntry } from "../types";
export function StatusBadge({
  open = true,
  label,
}: {
  open?: boolean;
  label?: string;
}) {
  return (
    <span className={`badge ${open ? "badge-green" : "badge-gray"}`}>
      <span className="status-dot" />
      {label || (open ? "Open now" : "Closed")}
    </span>
  );
}
export function PriorityBadge() {
  return <span className="badge badge-gold">★ Priority</span>;
}
export function BusinessIcon({ business }: { business: Business }) {
  const Icon =
    business.id === "northside"
      ? Scissors
      : business.id === "greencare"
        ? HeartPulse
        : business.id === "quickfix"
          ? Wrench
          : Sparkles;
  return (
    <div className={`business-icon ${business.color}`}>
      <Icon size={26} strokeWidth={1.5} />
    </div>
  );
}
export function BusinessCard({
  business,
  waiting,
}: {
  business: Business;
  waiting: number;
}) {
  return (
    <article className="card business-card">
      <div className="flex items-center justify-between">
        <BusinessIcon business={business} />
        <StatusBadge open={business.open} />
      </div>
      <p className="eyebrow mt-6">{business.category}</p>
      <h3>{business.name}</h3>
      <p className="muted location">
        <MapPin size={14} />
        {business.location}
      </p>
      <div className="business-stats">
        <span>
          <strong>{waiting}</strong> waiting
        </span>
        <span>
          <Clock3 size={14} />
          {business.open ? `${business.wait} min` : "Opens tomorrow"}
        </span>
      </div>
      <Link className="business-link" to={`/business/${business.id}`}>
        View business <ArrowRight size={17} />
      </Link>
    </article>
  );
}
export function CounterCard({
  id,
  entry,
  customer = false,
}: {
  id: number;
  entry?: QueueEntry;
  customer?: boolean;
}) {
  const content = (
    <>
      <div className="flex items-center justify-between">
        <span className="muted">Counter {id}</span>
        <span className={`status-dot ${entry ? "" : "inactive"}`} />
      </div>
      <strong className="counter-number">{entry?.id || "Available"}</strong>
      <span className="caption">
        {entry ? "Currently serving" : "Ready for the next customer"}
      </span>
      {!customer && (
        <span className="counter-link">
          Open workspace <ArrowRight size={15} />
        </span>
      )}
    </>
  );
  return customer ? (
    <div className="card counter-card">{content}</div>
  ) : (
    <Link className="card counter-card" to={`/counter/${id}`}>
      {content}
    </Link>
  );
}
export function EmptyState({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="card empty-state">
      <h3>{title}</h3>
      <p className="muted">{description}</p>
      <Link className="button secondary mt-5" to="/">
        Explore businesses <ArrowRight size={16} />
      </Link>
    </div>
  );
}
