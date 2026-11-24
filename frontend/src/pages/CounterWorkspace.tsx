import { Check, Monitor, RefreshCw, SkipForward, Users } from 'lucide-react';
import { AnimatePresence, motion } from 'motion/react';
import { useWorkspaceBusiness } from '../components/ApplicationShell';
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
	callNextQueue,
	getCounter,
	processCalledQueue,
	removeQueueFromCounter,
	skipQueue,
} from "../api/counters";
import { getBusinessQueues } from "../api/operations";
import { businessQueueEventsURL } from "../api/queues";
import { subscribeSSE } from "../api/sse";
import { AccountAccess } from "../components/AccountAccess";
import {
	Button,
	Card,
	InlineError,
	PageContainer,
	StatusBadge,
} from "../components/foundation";
import { LiveValue } from "../components/LiveMotion";
import { useToast } from "../components/Toast";
import { OperationsFeedback } from "./BusinessOperations";
import { isBusinessId } from "./businessDisplay";
import { PriorityBadge } from "./Dashboard";
import {
	currentQueue,
	elapsedServiceTime,
	operationsError,
} from "./operationsDisplay";
import { useOperations } from "./useOperations";

export default function CounterWorkspace() {
	const { counterId = "" } = useParams();
	return (
		<PageContainer className="ql-counter-workspace-page">
			<h1>Counter workspace</h1>
			<AccountAccess>
				{isBusinessId(counterId) ? (
					<Workspace key={counterId} counterId={counterId} />
				) : (
					<Card>
						<InlineError>This counter link is invalid.</InlineError>
					</Card>
				)}
			</AccountAccess>
		</PageContainer>
	);
}

function ServiceTimer({ startedAt }: { startedAt?: string }) {
	const [now, setNow] = useState(Date.now());
	useEffect(() => {
		if (!startedAt) return;
		setNow(Date.now());
		const timer = window.setInterval(() => setNow(Date.now()), 1000);
		return () => window.clearInterval(timer);
	}, [startedAt]);
	return (
		<output className="ql-service-timer" aria-label="Time taken">
			{startedAt ? elapsedServiceTime(startedAt, now) : "Timing unavailable"}
		</output>
	);
}

function Workspace({ counterId }: { counterId: string }) {
	const load = useCallback(
		async (signal: AbortSignal) => {
			const counter = await getCounter(counterId, { signal });
			const queues = await getBusinessQueues(counter.businessId, { signal });
			return { counter, queues };
		},
		[counterId],
	);
	const resource = useOperations(load, false);
	const toast = useToast();
	const [busy, setBusy] = useState(false),
		[error, setError] = useState(""),
		[notice, setNotice] = useState(""),
		[uncertain, setUncertain] = useState(false),
		[refreshTick, setRefreshTick] = useState(0);
	const lock = useRef(false);
	const data = resource.data,
		queue = data && currentQueue(data.counter, data.queues);
	const businessId = data?.counter.businessId;
  useWorkspaceBusiness(businessId);

	useEffect(() => {
		if (!businessId) return;
		return subscribeSSE({
			url: businessQueueEventsURL(businessId),
			event: "queue.update",
			onEvent: () => {
				if (!lock.current) resource.retry();
			},
		});
	}, [businessId, resource.retry]);

	async function act(action: "call" | "process" | "complete" | "skip") {
		if (lock.current || !data || uncertain) return;
		lock.current = true;
		setBusy(true);
		setError("");
		setNotice("");
		try {
			if (action === "call") {
				const result = await callNextQueue(counterId, data.counter.businessId);
				if ("queue" in result && result.queue === null)
					setNotice("No waiting customer available.");
				else toast.success("Next customer called.");
			} else if (queue) {
				if (action === "process") {
					await processCalledQueue(counterId, queue.id);
					toast.info(`Started serving ${queue.name}.`);
				} else if (action === "complete") {
					await removeQueueFromCounter(counterId, queue.id);
					toast.success(`Completed visit for ${queue.name}.`);
				} else {
					await skipQueue(counterId, queue.id);
					setNotice(
						"Customer skipped. The scheduler selected the next available customer.",
					);
				}
			}
		} catch (failure) {
			setError(operationsError(failure));
			setUncertain(true);
		} finally {
			lock.current = false;
			setBusy(false);
			// refresh the waiting counter
			resource.retry();
		}
	}

	function refresh() {
		setError("");
		setNotice("");
		setUncertain(false);
		setRefreshTick((v) => v + 1);
		resource.retry();
	}
	const active =
		queue && (queue.state === "called" || queue.state === "processing");
	const waiting =
		data?.queues.filter((item) => item.state === "waiting").length ?? 0;

	const actRef = useRef(act);
	actRef.current = act;

	// Single-key operational shortcuts on desktop when focus is not inside an editable control.
	useEffect(() => {
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.ctrlKey || event.metaKey || event.altKey || busy || uncertain || !data) return;
			const target = event.target as HTMLElement | null;
			if (target && (["INPUT", "TEXTAREA", "SELECT", "BUTTON", "A"].includes(target.tagName) || target.isContentEditable)) return;
			const key = event.key.toLowerCase();
			if (!active && !data.counter.currentQueueId && key === "c") {
				event.preventDefault();
				void actRef.current("call");
			} else if (active && queue?.state === "called" && key === "s") {
				event.preventDefault();
				void actRef.current("process");
			} else if (active && queue?.state === "processing" && key === "d") {
				event.preventDefault();
				void actRef.current("complete");
			} else if (active && key === "k") {
				event.preventDefault();
				void actRef.current("skip");
			}
		};
		window.addEventListener("keydown", onKeyDown);
		return () => window.removeEventListener("keydown", onKeyDown);
	}, [active, busy, data, queue?.state, uncertain]);

	const workspaceTone = active
		? queue.state === "processing"
			? "ql-workspace-state-processing"
			: "ql-workspace-state-called"
		: "ql-workspace-state-idle";
	const ticketViewKey = active
		? `${queue.id}-${queue.state}`
		: data?.counter.currentQueueId
			? "pending"
			: "idle";

	return (
		<div className="ql-stack">
			<OperationsFeedback {...resource} />
			{error && (
				<Card>
					<InlineError>{error}</InlineError>
					<Button variant="secondary" disabled={busy} onClick={refresh}>
						Refresh state
					</Button>
				</Card>
			)}
			{notice && <p role="status" className="ql-success-note">{notice}</p>}
			{data && (
				<>
					<Link
						className="ql-back-link"
						to={`/business/${data.counter.businessId}/dashboard`}
					>
						← Business dashboard
					</Link>
					<Card className="ql-workspace-header">
						<div>
							<p className="ql-eyebrow">Active service counter</p>
							<h2>{data.counter.name}</h2>
							<p className="ql-muted">Waiting customers: <LiveValue value={String(waiting)} /></p>
						</div>
						<Button variant="secondary" disabled={busy} onClick={refresh}>
							<RefreshCw key={refreshTick} className={refreshTick > 0 ? "ql-spin-once" : undefined} size={15} aria-hidden="true" />
							Refresh workspace
						</Button>
					</Card>
					<div className="ql-workspace-grid">
						<Card
							className={`ql-workspace-current ${workspaceTone}`}
							aria-labelledby="current-customer-heading"
						>
							<h2 id="current-customer-heading">Currently serving</h2><Monitor className="ql-workspace-icon" size={26} aria-hidden="true" />
							<AnimatePresence mode="wait" initial={false}>
								<motion.div
									key={ticketViewKey}
									initial={{ opacity: 0.25, x: 16 }}
									animate={{ opacity: 1, x: 0 }}
									exit={{ opacity: 0, x: -16 }}
									transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
								>
									{active ? (
										<>
											<div className="ql-workspace-badges">
												<StatusBadge state={queue.state} />
												<PriorityBadge queue={queue} />
											</div>
											<p className="ql-queue-number">{queue.name}</p>
											{queue.state === "processing" ? (
												<div className="ql-time-panel">
													<span className="ql-meta ql-timer-label"><span className="ql-timer-dot" aria-hidden="true" />Time taken</span>
													<ServiceTimer startedAt={queue.processingAt} />
												</div>
											) : (
												<p className="ql-muted">
													Service time starts after you begin service.
												</p>
											)}
											<div className="ql-actions">
												{queue.state === "called" ? (
													<Button
														loading={busy}
														disabled={uncertain}
														onClick={() => void act("process")}
													>
														Start service <kbd className="ql-kbd ql-kbd-on-dark" aria-hidden="true">S</kbd>
													</Button>
												) : (
													<Button
														loading={busy}
														disabled={uncertain}
														onClick={() => void act("complete")}
													>
														<Check size={17} aria-hidden="true" /> Complete <kbd className="ql-kbd ql-kbd-on-dark" aria-hidden="true">D</kbd>
													</Button>
												)}
												<Button
													variant="secondary"
													disabled={busy || uncertain}
													onClick={() => void act("skip")}
												>
													<SkipForward size={17} aria-hidden="true" /> Skip <kbd className="ql-kbd" aria-hidden="true">K</kbd>
												</Button>
											</div>
										</>
									) : data.counter.currentQueueId ? (
										<>
											<h3>Customer state pending</h3>
											<p className="ql-muted">
												The assigned queue is not available in an active state.
												Refresh before operating this counter.
											</p>
										</>
									) : (
										<>
											<p className="ql-workspace-empty">No active customer</p>
											<Button
												loading={busy}
												disabled={uncertain}
												onClick={() => void act("call")}
											>
												Call next <kbd className="ql-kbd ql-kbd-on-dark" aria-hidden="true">C</kbd>
											</Button>
										</>
									)}
								</motion.div>
							</AnimatePresence>
						</Card>
						<Card
							className="ql-workspace-next"
							aria-labelledby="next-customer-heading"
						>
							<h2 id="next-customer-heading">Next customer</h2>
              <Users className="ql-workspace-icon" size={27} aria-hidden="true" /><p className="ql-workspace-next-value"><LiveValue value={`${waiting} ${waiting === 1 ? 'customer' : 'customers'} waiting`} /></p>
							<p className="ql-muted">
								Call the next customer when this counter is ready.
							</p>
              <p className="ql-meta">Complete finishes this visit. Skip also calls the next waiting customer.</p>
						</Card>
					</div>
				</>
			)}
		</div>
	);
}
