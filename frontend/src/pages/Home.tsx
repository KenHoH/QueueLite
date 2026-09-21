import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { ArrowRight, Search, Clock3, Ticket, ArrowUpRight } from "lucide-react";
import { businesses } from "../data/mock";
import { useQueue } from "../state/QueueContext";
import { BusinessCard, StatusBadge } from "../components/ui";
export default function Home() {
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState("All businesses");
  const { entries } = useQueue();
  const { hash } = useLocation();
  useEffect(() => {
    if (hash)
      document
        .getElementById(hash.slice(1))
        ?.scrollIntoView({ behavior: "smooth" });
  }, [hash]);
  const mine = entries.filter(
    (e) => e.mine && ["Waiting", "Serving"].includes(e.status),
  );
  const filtered = businesses.filter(
    (b) =>
      `${b.name} ${b.location} ${b.category}`
        .toLowerCase()
        .includes(search.toLowerCase()) &&
      (category === "All businesses" || b.category === category),
  );
  return (
    <>
      <section className="home-hero">
        <div>
          <div className="eyebrow greeting-label">
            <span className="small-line" /> MORE LIFE. LESS WAITING.
          </div>
          <h1>
            Good afternoon, Matthew<span className="mint">.</span>
          </h1>
          <p className="lead">
            Find a business and join the queue without waiting in line.
          </p>
        </div>
        <div className="hero-note">
          <span className="mini-logo">
            <Clock3 size={21} />
          </span>
          <span>
            Your time.
            <br />
            <strong>Back in your hands.</strong>
          </span>
        </div>
      </section>
      <div className="search-field">
        <Search size={21} />
        <input
          aria-label="Search businesses or services"
          placeholder="Search businesses or services..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <span className="search-location">
          <span className="status-dot" /> Jakarta, Indonesia
        </span>
      </div>
      <section id="my-queues" className="section">
        <div className="section-heading">
          <h2>
            Your active queues <span className="count">{mine.length}</span>
          </h2>
          <span className="caption">A little freedom while you wait</span>
        </div>
        {mine.length ? (
          mine.map((entry) => {
            const business = businesses.find((b) => b.id === entry.businessId)!;
            const waiting = entries.filter(
              (e) => e.businessId === business.id && e.status === "Waiting",
            );
            const ahead = Math.max(
              0,
              waiting.findIndex((e) => e.id === entry.id),
            );
            return (
              <article className="active-queue" key={entry.id}>
                <div className="active-business">
                  <span className="active-icon">
                    <Ticket size={23} />
                  </span>
                  <div>
                    <StatusBadge
                      label={
                        entry.status === "Serving"
                          ? "It's your turn"
                          : "You're in line"
                      }
                    />
                    <h3>{business.name}</h3>
                    <p className="muted">
                      {business.location} <span className="footer-dot">·</span>{" "}
                      {entry.service} service
                    </p>
                  </div>
                </div>
                <div className="ticket-number">
                  <span className="eyebrow">YOUR NUMBER</span>
                  <strong>{entry.id}</strong>
                </div>
                <div className="queue-estimate">
                  <span>
                    Now serving{" "}
                    {entries.find(
                      (e) =>
                        e.businessId === business.id && e.status === "Serving",
                    )?.id || "—"}
                  </span>
                  <strong>
                    {entry.status === "Serving"
                      ? `Counter ${entry.counter}`
                      : `~${Math.max(5, ahead * 5)} min`}
                  </strong>
                  <span>
                    {entry.status === "Serving"
                      ? "Please head to your counter"
                      : `Approximately ${ahead} customers ahead`}
                  </span>
                </div>
                <Link className="button primary" to={`/queue/${entry.id}`}>
                  View queue <ArrowRight size={17} />
                </Link>
              </article>
            );
          })
        ) : (
          <div className="card empty-state">
            <Ticket size={25} />
            <h3>A little more time for you.</h3>
            <p className="muted">
              Find a business below to join your first queue.
            </p>
          </div>
        )}
      </section>
      <section id="businesses" className="section">
        <div className="section-heading">
          <div>
            <h2>Discover your neighborhood</h2>
            <p className="muted mt-2">
              Great local businesses. A better way to wait.
            </p>
          </div>
          <span className="caption">
            {filtered.length} businesses available
          </span>
        </div>
        <div className="filter-row" aria-label="Business categories">
          {[
            "All businesses",
            "Barbershop",
            "Healthcare",
            "Repair & service",
            "Beauty & wellness",
          ].map((c) => (
            <button
              key={c}
              className={`filter-chip ${category === c ? "selected" : ""}`}
              onClick={() => setCategory(c)}
            >
              {c}
            </button>
          ))}
        </div>
        <div className="business-grid">
          {filtered.map((b) => (
            <BusinessCard
              key={b.id}
              business={b}
              waiting={
                entries.filter(
                  (e) => e.businessId === b.id && e.status === "Waiting",
                ).length
              }
            />
          ))}
        </div>
        {!filtered.length && (
          <div className="empty-state card">
            <h3>No businesses found</h3>
            <p className="muted">Try a different name, service, or category.</p>
            <button
              className="button secondary mt-4"
              onClick={() => {
                setSearch("");
                setCategory("All businesses");
              }}
            >
              Clear filters
            </button>
          </div>
        )}
      </section>
      <div className="home-bottom">
        <span>
          <span className="mint">✦</span> Step out for a coffee. We'll keep your
          place in line.
        </span>
        <Link to="/dashboard">
          Explore the business demo <ArrowUpRight size={15} />
        </Link>
      </div>
    </>
  );
}
