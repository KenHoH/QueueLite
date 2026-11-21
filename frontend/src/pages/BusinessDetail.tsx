import {
	ArrowLeft,
	ArrowRight,
	Clock3,
	Mail,
	MapPin,
	Phone,
	Check,
	Store,
	Ticket,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { getBusiness } from "../api/businesses";
import { APIError } from "../api/errors";
import type { Business } from "../api/types";
import { BusinessAvailability } from "../components/BusinessCard";
import {
	Button,
	Card,
	EmptyState,
	InlineError,
	LoadingSkeleton,
	PageContainer,
} from "../components/foundation";
import { businessHours, isBusinessId } from "./businessDisplay";

type DetailState =
	| {
			[Status in "loading" | "not-found" | "error"]: {
				id: string | undefined;
				status: Status;
			};
	  }["loading" | "not-found" | "error"]
	| { id: string; status: "ready"; business: Business };
export default function BusinessDetail() {
	const { businessId } = useParams();
	const [attempt, setAttempt] = useState(0);
	const [state, setState] = useState<DetailState>({
		id: businessId,
		status: "loading",
	});
	useEffect(() => {
		if (!isBusinessId(businessId)) {
			setState({ id: businessId, status: "not-found" });
			return;
		}
		const controller = new AbortController();
		setState({ id: businessId, status: "loading" });
		void getBusiness(businessId, { signal: controller.signal })
			.then((business) => {
				if (!controller.signal.aborted)
					setState({ id: businessId, status: "ready", business });
			})
			.catch((error) => {
				if (!controller.signal.aborted)
					setState({
						id: businessId,
						status:
							error instanceof APIError &&
							(error.status === 404 ||
								error.code === "BUSINESS_NOT_FOUND" ||
								error.code === "INVALID_BUSINESS_ID")
								? "not-found"
								: "error",
					});
			});
		return () => controller.abort();
	}, [businessId, attempt]);
	const current =
		state.id === businessId ? state : { status: "loading" as const };
	const back = (
		<Link className="ql-back-link" to="/#businesses">
			<ArrowLeft size={16} aria-hidden="true" />
			Back to businesses
		</Link>
	);
	return <PageContainer className="ql-business-detail">
    {back}
    {current.status === 'loading' ? <Card><LoadingSkeleton label="Loading business details" lines={7} /></Card>
      : current.status === 'not-found' ? <EmptyState title="Business not found." description="This business may no longer be available." action={<Link to="/#businesses">Back to businesses</Link>} />
      : current.status === 'error' ? <Card><InlineError>We couldn’t load this business.</InlineError><Button variant="secondary" onClick={() => setAttempt(value => value + 1)}>Try again</Button></Card>
      : <>
        <div className="ql-business-banner"><div><p className="ql-eyebrow">Your neighborhood, at your pace</p><p>More than a visit. A little time for you.</p></div><span aria-hidden="true">Q<span>Lite</span></span></div>
        <div className="ql-detail-layout">
          <section className="ql-detail-content">
            <div className="ql-business-title"><span className="ql-business-icon" aria-hidden="true"><Store size={30} /></span><div><BusinessAvailability business={current.business} /><h1>{current.business.name}</h1><p className="ql-business-location"><MapPin size={17} aria-hidden="true" />{current.business.location || 'Location not provided'}</p></div></div>
            <div className="ql-detail-description"><h2>About this business</h2><p className="ql-muted">{current.business.description || 'No description provided.'}</p></div>
            <dl className="ql-contact-list">
              <div><dt><Clock3 size={17} aria-hidden="true" /><span className="ql-sr-only">Daily hours</span></dt><dd>{businessHours(current.business)}</dd></div>
              <div><dt><Mail size={17} aria-hidden="true" /><span className="ql-sr-only">Email</span></dt><dd>{current.business.email ? <a href={`mailto:${current.business.email}`}>{current.business.email}</a> : 'Not provided'}</dd></div>
              <div><dt><Phone size={17} aria-hidden="true" /><span className="ql-sr-only">Phone</span></dt><dd>{current.business.phoneNumber ? <a href={`tel:${current.business.phoneNumber.replace(/[^+\d]/g, '')}`}>{current.business.phoneNumber}</a> : 'Not provided'}</dd></div>
            </dl>
            <p className="ql-soft-note"><Check size={18} aria-hidden="true" />Join from wherever you are. Keep an eye on your ticket and head over when your turn is close.</p>
          </section>
          <Card asChild className="ql-business-information"><aside>
            <h2>Business information</h2>
            <div className="ql-join-preview"><Ticket size={28} aria-hidden="true" /><h3>A little less waiting.</h3><p>Save your place, then follow your ticket for updates.</p></div>
            {current.business.operational ? <Button asChild className="ql-join-link"><Link to={`/business/${encodeURIComponent(current.business.id)}/join`}>Join queue <ArrowRight size={17} aria-hidden="true" /></Link></Button> : <Button className="ql-join-link" disabled>Join queue</Button>}
            <p className="ql-meta ql-join-note">{current.business.operational ? 'No standing around. No account needed.' : 'This business is currently unavailable.'}</p>
          </aside></Card>
        </div>
      </>}
  </PageContainer>;
}
