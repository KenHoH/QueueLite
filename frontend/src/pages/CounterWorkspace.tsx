import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  SkipForward,
  X,
  Clock3,
  Users,
  Monitor,
} from "lucide-react";
import { useQueue } from "../state/QueueContext";
import { counters } from "../data/mock";
import { EmptyState, PriorityBadge, StatusBadge } from "../components/ui";
export default function CounterWorkspace() {
  const { id } = useParams();
  const counter = counters.find((c) => c.id === Number(id));
  const { entries, advance } = useQueue();
  const [notice, setNotice] = useState("");
  if (!counter)
    return (
      <EmptyState
        title="Counter not found"
        description="Open a counter from the business dashboard."
      />
    );
  const current = entries.find(
    (e) =>
      e.businessId === "northside" &&
      e.counter === counter.id &&
      e.status === "Serving",
  );
  const waiting = entries.filter(
    (e) => e.businessId === "northside" && e.status === "Waiting",
  );
  const next = waiting[0];
  function act(outcome?: "Completed" | "Skipped" | "Cancelled") {
    advance(counter!.id, outcome);
    setNotice(
      `${current ? `${current.id} ${outcome?.toLowerCase()}. ` : ""}${next ? `${next.id} called to Counter ${counter!.id}.` : "No customers waiting."}`,
    );
  }
  return (
    <>
      <Link to="/dashboard" className="back-link">
        <ArrowLeft size={16} /> Back to overview
      </Link>
      <div className="dashboard-heading">
        <div>
          <span className="eyebrow">NORTHSIDE BARBERS</span>
          <h1>Counter {counter.id}</h1>
          <p className="muted">
            One customer at a time. A great experience for everyone.
          </p>
        </div>
        <span className="operator">
          <span className="avatar">{counter.operator.slice(0, 1)}</span>
          <span>
            <strong>{counter.operator}</strong>
            <small>Counter operator</small>
          </span>
        </span>
      </div>
      <div className="workspace-layout">
        <section className="card serving-workspace">
          <div className="flex items-center justify-between">
            <span className="eyebrow">CURRENTLY SERVING</span>
            <StatusBadge label={current ? "In service" : "Available"} />
          </div>
          <div className="workspace-ticket">
            <Monitor size={25} />
            <strong>{current?.id || "Ready"}</strong>
            <h3>{current?.name || "Ready for a new customer"}</h3>
            <p className="muted">
              {current
                ? `${current.service} service`
                : "Call the next person when you are ready."}
            </p>
            {current?.priority && <PriorityBadge />}
            <span className="started">
              <Clock3 size={14} />
              {current
                ? `Service started: ${current.startedAt}`
                : "No active service"}
            </span>
          </div>
          <button
            className="button primary full complete-button"
            disabled={!current}
            onClick={() => act("Completed")}
          >
            <Check size={19} /> Complete & call next
          </button>
          <div className="workspace-actions">
            <button
              className="button secondary"
              disabled={!current}
              onClick={() => act("Skipped")}
            >
              <SkipForward size={17} /> Skip
            </button>
            <button
              className="button danger"
              disabled={!current}
              onClick={() => act("Cancelled")}
            >
              <X size={17} /> Cancel
            </button>
          </div>
          <p className="caption text-center mt-4">
            Finishing a service automatically calls the next customer.
          </p>
        </section>
        <aside>
          <div className="card next-workspace">
            <span className="eyebrow">UP NEXT</span>
            <strong>{next?.id || "All caught up"}</strong>
            {next?.priority ? (
              <PriorityBadge />
            ) : (
              <span className="badge badge-gray">
                {next ? "Standard" : "Queue empty"}
              </span>
            )}
            <p className="muted">
              {next
                ? `${next.name} · ${next.service}`
                : "Take a moment. There is no one waiting."}
            </p>
            <button
              className="button secondary full"
              disabled={!!current || !next}
              onClick={() => act()}
            >
              Call next <ArrowRight size={17} />
            </button>
            {current && (
              <p className="caption mt-3">
                Finish the current service before calling next.
              </p>
            )}
          </div>
          <div className="card remaining-card">
            <Users size={22} />
            <div>
              <strong>{waiting.length} customers</strong>
              <span>Remaining in the queue</span>
            </div>
          </div>
          <div className="operator-note">
            <h3>A focused workspace.</h3>
            <p>
              Everything you need for this counter. For all counters and queue
              details, return to the overview.
            </p>
          </div>
        </aside>
      </div>
      <div
        role="status"
        aria-live="polite"
        className={notice ? "toast" : "sr-only"}
      >
        {notice}
      </div>
    </>
  );
}
