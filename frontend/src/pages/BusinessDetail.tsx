import { Link, useParams } from "react-router-dom";
import {
  ArrowLeft,
  ArrowRight,
  Clock3,
  MapPin,
  Mail,
  Phone,
  Users,
  Monitor,
  Check,
} from "lucide-react";
import { businesses } from "../data/mock";
import { BusinessIcon, EmptyState, StatusBadge } from "../components/ui";
import { useQueue } from "../state/QueueContext";
export default function BusinessDetail() {
  const { id } = useParams();
  const business = businesses.find((b) => b.id === id);
  const { entries } = useQueue();
  if (!business)
    return (
      <EmptyState
        title="Business not found"
        description="Choose one of the businesses in our neighborhood."
      />
    );
  const waiting = entries.filter(
    (e) => e.businessId === id && e.status === "Waiting",
  ).length;
  return (
    <>
      <Link to="/#businesses" className="back-link">
        <ArrowLeft size={16} /> Back to businesses
      </Link>
      <div className="detail-banner">
        <span className="eyebrow">YOUR NEIGHBORHOOD, AT YOUR PACE</span>
        <span className="banner-mark">N° {business.initials}</span>
        <span>More than a service. A little time for you.</span>
      </div>
      <div className="detail-layout">
        <section>
          <div className="detail-identity">
            <BusinessIcon business={business} />
            <div>
              <div className="flex items-center gap-3">
                <span className="eyebrow">{business.category}</span>
                <StatusBadge open={business.open} />
              </div>
              <h1>{business.name}</h1>
              <p className="muted location">
                <MapPin size={16} />
                {business.location}
              </p>
            </div>
          </div>
          <div className="detail-about">
            <h2>A good visit starts here.</h2>
            <p className="lead">{business.description}</p>
            <div className="contact-list">
              <span>
                <MapPin size={18} />
                {business.address}
              </span>
              <a href={`mailto:${business.email}`}>
                <Mail size={18} />
                {business.email}
              </a>
              <a href={`tel:${business.phone.replace(/\s/g, "")}`}>
                <Phone size={18} />
                {business.phone}
              </a>
            </div>
          </div>
          <div className="info-note">
            <Check size={19} />
            <p>
              Join from wherever you are. Keep an eye on your queue and head
              over when your turn is close.
            </p>
          </div>
        </section>
        <aside className="card queue-summary">
          <div className="section-heading">
            <h2>The queue right now</h2>
            <span className="status-dot" />
          </div>
          <div className="summary-wait">
            <Clock3 size={21} />
            <strong>{business.open ? business.wait : "—"}</strong>
            <span>minutes estimated wait</span>
          </div>
          <div className="summary-line">
            <span>
              <Users size={17} />
              Customers waiting
            </span>
            <strong>{waiting}</strong>
          </div>
          <div className="summary-line">
            <span>
              <Monitor size={17} />
              Active counters
            </span>
            <strong>{business.counters}</strong>
          </div>
          {business.open ? (
            <Link to={`/business/${id}/join`} className="button primary full">
              Join queue <ArrowRight size={17} />
            </Link>
          ) : (
            <button className="button primary full" disabled>
              Currently closed
            </button>
          )}
          <p className="caption text-center mt-4">
            {business.open
              ? "No standing around. No account needed."
              : "Please check back during opening hours."}
          </p>
        </aside>
      </div>
    </>
  );
}
