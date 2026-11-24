import { useEffect, useRef, useState } from "react";

/**
 * Crisp geometric SVG analog clock displaying the visitor's real local time.
 * Pauses ticking when the tab is hidden to stay lightweight.
 */
export function LiveClock({ size = 24 }: { size?: number }) {
	const [now, setNow] = useState(() => new Date());
	const secTurns = useRef(0);
	const prevSec = useRef(now.getSeconds());

	useEffect(() => {
		let timer: number | undefined;
		const tick = () => {
			const next = new Date();
			const s = next.getSeconds();
			if (s < prevSec.current) secTurns.current += 1;
			prevSec.current = s;
			setNow(next);
		};
		const start = () => {
			if (document.hidden || timer !== undefined) return;
			tick();
			timer = window.setInterval(tick, 1000);
		};
		const stop = () => {
			if (timer !== undefined) {
				window.clearInterval(timer);
				timer = undefined;
			}
		};
		const onVisibility = () => {
			if (document.hidden) stop();
			else start();
		};
		start();
		document.addEventListener("visibilitychange", onVisibility);
		return () => {
			stop();
			document.removeEventListener("visibilitychange", onVisibility);
		};
	}, []);

	const seconds = now.getSeconds();
	const minutes = now.getMinutes() + seconds / 60;
	const hours = (now.getHours() % 12) + minutes / 60;

	const hourDeg = hours * 30;
	const minDeg = minutes * 6;
	const secDeg = secTurns.current * 360 + seconds * 6;

	return (
		<svg
			className="ql-live-clock"
			width={size}
			height={size}
			viewBox="0 0 24 24"
			fill="none"
			aria-hidden="true"
		>
			<circle
				cx="12"
				cy="12"
				r="9.5"
				stroke="currentColor"
				strokeWidth="1.75"
			/>
			<line
				x1="12"
				y1="12"
				x2="12"
				y2="7.2"
				stroke="currentColor"
				strokeWidth="1.9"
				strokeLinecap="round"
				style={{ transform: `rotate(${hourDeg}deg)`, transformOrigin: "12px 12px" }}
			/>
			<line
				x1="12"
				y1="12"
				x2="12"
				y2="5.4"
				stroke="currentColor"
				strokeWidth="1.5"
				strokeLinecap="round"
				style={{ transform: `rotate(${minDeg}deg)`, transformOrigin: "12px 12px" }}
			/>
			<line
				className="ql-clock-second"
				x1="12"
				y1="13.6"
				x2="12"
				y2="4.9"
				stroke="var(--ql-mint)"
				strokeWidth="1.15"
				strokeLinecap="round"
				style={{ transform: `rotate(${secDeg}deg)`, transformOrigin: "12px 12px" }}
			/>
			<circle cx="12" cy="12" r="1.2" fill="currentColor" />
		</svg>
	);
}

/**
 * Minimal geometric queue-advancing vector loop showing tickets moving calmly
 * toward an open service counter. Pure CSS animation with reduced-motion fallback.
 */
export function QueueFlowVisual() {
	return (
		<svg
			className="ql-queue-flow-svg"
			width="112"
			height="30"
			viewBox="0 0 112 30"
			fill="none"
			aria-hidden="true"
		>
			<line
				x1="14"
				y1="15"
				x2="86"
				y2="15"
				stroke="var(--ql-mint-border)"
				strokeWidth="1.5"
				strokeDasharray="3 4"
			/>
			<rect
				className="ql-flow-ticket ql-flow-ticket-1"
				x="8"
				y="9"
				width="16"
				height="12"
				rx="3.5"
				fill="var(--ql-surface)"
				stroke="var(--ql-mint)"
				strokeWidth="1.5"
			/>
			<rect
				className="ql-flow-ticket ql-flow-ticket-2"
				x="34"
				y="9"
				width="16"
				height="12"
				rx="3.5"
				fill="var(--ql-surface)"
				stroke="var(--ql-mint)"
				strokeWidth="1.5"
			/>
			<rect
				className="ql-flow-ticket ql-flow-ticket-3"
				x="60"
				y="9"
				width="16"
				height="12"
				rx="3.5"
				fill="var(--ql-soft-mint)"
				stroke="var(--ql-action)"
				strokeWidth="1.5"
			/>
			<circle
				className="ql-flow-counter"
				cx="96"
				cy="15"
				r="10"
				fill="var(--ql-action)"
			/>
			<path
				d="M92 15.2l2.6 2.6 5.4-5.6"
				stroke="var(--ql-surface)"
				strokeWidth="1.9"
				strokeLinecap="round"
				strokeLinejoin="round"
			/>
		</svg>
	);
}
