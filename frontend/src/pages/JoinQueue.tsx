import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  Check,
  Clock3,
  ShieldCheck,
} from "lucide-react";
import { businesses } from "../data/mock";
import { BusinessIcon, EmptyState } from "../components/ui";
import { useQueue } from "../state/QueueContext";
export default function JoinQueue() {
  const { id } = useParams();
  const business = businesses.find((b) => b.id === id);
  const { entries, join } = useQueue();
  const [name, setName] = useState("Matthew Sutiono");
  const [phone, setPhone] = useState("+62 812 3456 7890");
  const [ticket, setTicket] = useState("");
  const [notify, setNotify] = useState(true);
  const [error, setError] = useState("");
  if (!business || !business.open)
    return (
      <EmptyState
        title="This queue is unavailable"
        description="Please choose an open business to join a queue."
      />
    );
  const existing = entries.find(
    (e) =>
      e.businessId === id &&
      e.mine &&
      ["Waiting", "Serving"].includes(e.status),
  );
  const waiting = entries.filter(
    (e) => e.businessId === id && e.status === "Waiting",
  );
  const ahead = ticket
    ? Math.max(
        0,
        waiting.findIndex((e) => e.id === ticket),
      )
    : waiting.length;
  if (ticket)
    return (
      <div className="success-layout">
        <div className="success-check">
          <Check size={31} />
        </div>
        <span className="eyebrow">A LITTLE LESS WAITING STARTS NOW</span>
        <h1>You're in!</h1>
        <p className="lead">Your place at {business.name} is saved.</p>
        <div className="card success-ticket">
          <span className="eyebrow">YOUR QUEUE NUMBER</span>
          <strong>{ticket}</strong>
          <div className="ticket-dash" />
          <p>Approximately {ahead} customers ahead</p>
          <span className="muted">
            {existing?.status === "Serving"
              ? "Your counter is ready"
              : `Estimated wait: ${Math.max(5, ahead * 3)}–${Math.max(10, ahead * 3 + 10)} minutes`}
          </span>
        </div>
        <Link to={`/queue/${ticket}`} className="button primary">
          View my queue <ArrowRight size={17} />
        </Link>
        <p className="caption mt-5">
          {notify
            ? "Notification preference saved for this demo; no alerts are sent."
            : "Check your queue page for your latest position."}
        </p>
      </div>
    );
  return (
    <>
      <Link className="back-link" to={`/business/${id}`}>
        <ArrowLeft size={16} /> Back to business
      </Link>
      <div className="flow-heading">
        <span className="eyebrow">YOUR NEXT STOP, WITHOUT THE WAIT</span>
        <h1>Let's save your place.</h1>
        <p className="lead">A few details, and you're good to go.</p>
      </div>
      <div className="join-layout">
        <form
          className="card join-form"
          onSubmit={(e) => {
            e.preventDefault();
            if (!name.trim() || phone.replace(/\D/g, "").length < 8) {
              setError(
                "Enter your name and a valid phone number (at least 8 digits).",
              );
              return;
            }
            setTicket(join(business.id, name.trim()));
          }}
        >
          <h2>Your details</h2>
          <p className="muted mt-2 mb-7">So the team knows who to welcome.</p>
          <label className="field-label" htmlFor="name">
            Full name
          </label>
          <input
            className="form-input"
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            maxLength={80}
            autoComplete="name"
          />
          <label className="field-label mt-6" htmlFor="phone">
            Phone number
          </label>
          <input
            className="form-input"
            id="phone"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            type="tel"
            required
            maxLength={24}
            autoComplete="tel"
          />
          <label className="check-label">
            <input
              type="checkbox"
              checked={notify}
              onChange={(e) => setNotify(e.target.checked)}
            />
            <span>
              Notify me when my turn is approaching.
              <small>Preview preference only. No messages are sent.</small>
            </span>
          </label>
          {error && (
            <p className="error-text" role="alert">
              {error}
            </p>
          )}
          {existing && (
            <div className="info-note mb-5">
              <Check size={18} />
              <p>
                You already have an active ticket here. Continue to view your
                saved place.
              </p>
            </div>
          )}
          <button className="button primary full" type="submit">
            {existing ? "Continue with my queue" : "Join queue"}{" "}
            <ArrowRight size={17} />
          </button>
          <p className="caption text-center mt-4">
            Your time is yours. We'll hold your place.
          </p>
        </form>
        <aside>
          <div className="card join-business">
            <BusinessIcon business={business} />
            <h3 className="mt-5">{business.name}</h3>
            <p className="muted mt-2">{business.location}</p>
            <div className="summary-line">
              <span>Service</span>
              <strong>General Service</strong>
            </div>
            <div className="summary-line">
              <span>Customers waiting</span>
              <strong>{waiting.length}</strong>
            </div>
            <div className="join-wait">
              <Clock3 size={20} />
              <div>
                <strong>{business.wait} minutes</strong>
                <span>Approximate waiting time</span>
              </div>
            </div>
          </div>
          <p className="aside-note">
            <ShieldCheck size={20} /> This is a local prototype. Your details
            aren't sent to a server.
          </p>
        </aside>
      </div>
    </>
  );
}
