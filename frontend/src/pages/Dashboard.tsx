import { useState } from "react";
import {
  CheckCheck,
  SkipForward,
  CircleX,
  Clock3,
  CalendarDays,
  Users,
} from "lucide-react";
import { useQueue } from "../state/QueueContext";
import { counters } from "../data/mock";
import { CounterCard } from "../components/ui";
import QueueTable from "../components/QueueTable";
export default function Dashboard() {
  const { entries, totals } = useQueue();
  const [service, setService] = useState("All services");
  const active = entries.filter(
    (e) =>
      e.businessId === "northside" && ["Waiting", "Serving"].includes(e.status),
  );
  const stats = [
    {
      label: "Customers served",
      value: totals.Completed,
      caption: "Visits completed today",
      Icon: CheckCheck,
    },
    {
      label: "Skipped",
      value: totals.Skipped,
      caption: "Customers not present",
      Icon: SkipForward,
    },
    {
      label: "Cancelled",
      value: totals.Cancelled,
      caption: "Visits cancelled today",
      Icon: CircleX,
    },
    {
      label: "Avg. service time",
      value: "~10",
      caption: "Minutes per customer · sample",
      Icon: Clock3,
    },
  ];
  return (
    <>
      <div className="dashboard-heading">
        <div>
          <span className="eyebrow">YOUR BUSINESS, AT A GLANCE</span>
          <h1>
            Good afternoon<span className="mint">.</span>
          </h1>
          <p className="muted">
            Here's how things are moving at Northside Barbers.
          </p>
        </div>
        <span className="date-chip">
          <CalendarDays size={16} /> Today <span>·</span> Demo day
        </span>
      </div>
      <div className="stat-grid">
        {stats.map(({ label, value, caption, Icon }) => (
          <div className="card stat-card" key={label}>
            <div className="flex justify-between items-center">
              <span>{label}</span>
              <Icon size={17} />
            </div>
            <strong>
              {value}
              <small>{label === "Avg. service time" ? " min" : ""}</small>
            </strong>
            <p>{caption}</p>
          </div>
        ))}
      </div>
      <section className="section">
        <div className="section-heading">
          <div>
            <h2>
              Your counters <span className="count">3</span>
            </h2>
            <p className="muted mt-2">
              A clear view of every service in progress.
            </p>
          </div>
          <span className="caption">
            Select a counter to open its workspace
          </span>
        </div>
        <div className="counter-grid">
          {counters.map((c) => (
            <CounterCard
              key={c.id}
              id={c.id}
              entry={active.find(
                (e) => e.counter === c.id && e.status === "Serving",
              )}
            />
          ))}
        </div>
      </section>
      <section className="section">
        <div className="section-heading">
          <h2>Queue overview</h2>
          <span className="waiting-count">
            <Users size={16} />
            {active.filter((e) => e.status === "Waiting").length} customers
            waiting
          </span>
        </div>
        <div
          className="service-tabs"
          role="tablist"
          aria-label="Filter queue by service"
        >
          {["All services", "General", "Haircut", "Consultation"].map((s) => (
            <button
              role="tab"
              aria-selected={service === s}
              key={s}
              className={service === s ? "selected" : ""}
              onClick={() => setService(s)}
            >
              {s}
            </button>
          ))}
        </div>
        <QueueTable
          key={service}
          entries={active.filter(
            (e) => service === "All services" || e.service === service,
          )}
        />
        <p className="caption mt-4">
          ★ Priority tickets are marked for staff. This prototype serves tickets
          in their displayed order.
        </p>
      </section>
    </>
  );
}
