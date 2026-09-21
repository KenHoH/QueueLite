import { Link, useParams } from "react-router-dom";
import { ArrowLeft, Clock3, Info, MapPin, Ticket } from "lucide-react";
import { businesses } from "../data/mock";
import { useQueue } from "../state/QueueContext";
import { CounterCard, EmptyState, StatusBadge } from "../components/ui";
export default function QueueStatus() {
  const { id } = useParams();
  const { entries } = useQueue();
  const ticket = entries.find((e) => e.id === id && e.mine);
  if (!ticket)
    return (
      <EmptyState
        title="No saved queue found"
        description="Demo queues reset on refresh. Join a business to save your place."
      />
    );
  const business = businesses.find((b) => b.id === ticket.businessId)!;
  const waiting = entries.filter(
    (e) => e.businessId === business.id && e.status === "Waiting",
  );
  const ahead = Math.max(
    0,
    waiting.findIndex((e) => e.id === id),
  );
  const done = !["Waiting", "Serving"].includes(ticket.status);
  return (
    <>
      <Link to="/#my-queues" className="back-link">
        <ArrowLeft size={16} /> My queues
      </Link>
      <div className="queue-page-heading">
        <div>
          <span className="eyebrow">A LITTLE TIME FOR YOURSELF</span>
          <h1>
            Your place is saved<span className="mint">.</span>
          </h1>
          <p className="muted location">
            <MapPin size={15} />
            {business.name} <span>·</span> {business.location}
          </p>
        </div>
        <StatusBadge
          label={
            done
              ? ticket.status
              : ticket.status === "Serving"
                ? "It's your turn"
                : "In the queue"
          }
          open={!done}
        />
      </div>
      <div className="queue-status-layout">
        <section className="card my-ticket">
          <span className="ticket-icon">
            <Ticket size={23} />
          </span>
          <span className="eyebrow">YOUR QUEUE</span>
          <strong className="huge-number">{ticket.id}</strong>
          <span className="service-tag">{ticket.service} service</span>
          <div className="ticket-dash" />
          <h3>
            {done
              ? `Visit ${ticket.status.toLowerCase()}`
              : ticket.status === "Serving"
                ? `Head to Counter ${ticket.counter}`
                : `Approximately ${ahead} customers ahead`}
          </h3>
          <p className="muted">
            {done
              ? "Thanks for using QueueLite."
              : ticket.status === "Serving"
                ? "The team is ready to welcome you."
                : "Relax a little. Your place is right here."}
          </p>
          <div className="ticket-wait">
            <Clock3 size={19} />
            <span>
              Estimated waiting time
              <strong>
                {done || ticket.status === "Serving"
                  ? "No wait"
                  : `${Math.max(5, ahead * 3)}–${Math.max(10, ahead * 3 + 10)} minutes`}
              </strong>
            </span>
          </div>
        </section>
        <section className="serving-panel">
          <div className="section-heading">
            <h2>Now serving</h2>
            <span className="caption">{business.counters} counters</span>
          </div>
          <div className="counter-grid customer-counters">
            {Array.from({ length: business.counters }, (_, i) => (
              <CounterCard
                key={i}
                id={i + 1}
                entry={entries.find(
                  (e) =>
                    e.businessId === business.id &&
                    e.status === "Serving" &&
                    e.counter === i + 1,
                )}
                customer
              />
            ))}
          </div>
          <div className="card up-next-card">
            <div>
              <span className="eyebrow">UP NEXT</span>
              <strong>{waiting[0]?.id || "No one waiting"}</strong>
            </div>
            <span className="caption">Next available counter</span>
          </div>
          <div className="info-note">
            <Info size={19} />
            <p>
              Queue position and estimated waiting time may change based on
              service conditions. Priority customers may be called sooner.
            </p>
          </div>
          <div className="while-waiting">
            <h3>Make the wait your own.</h3>
            <p className="muted">
              Grab a coffee, take a short walk, or finish that last email. Head
              back when your turn is close.
            </p>
            <Link to={`/business/${business.id}`} className="text-link">
              View business details →
            </Link>
          </div>
        </section>
      </div>
    </>
  );
}
