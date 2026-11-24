import { AnimatePresence, motion } from "motion/react";
import { useEffect, useRef, useState } from "react";

interface LiveValueProps {
	value: string;
	className?: string;
}

/**
 * Animates live SSE text changes with a subtle vertical roll + mint flash
 * while keeping the full string intact in a single accessible node.
 */
export function LiveValue({ value, className }: LiveValueProps) {
	const previous = useRef(value);
	const [flashing, setFlashing] = useState(false);

	useEffect(() => {
		if (previous.current === value) return;
		previous.current = value;
		setFlashing(true);
		const timer = window.setTimeout(() => setFlashing(false), 900);
		return () => window.clearTimeout(timer);
	}, [value]);

	return (
		<span
			className={`ql-live-value-wrap ${flashing ? "is-flashing" : ""} ${className ?? ""}`}
		>
			<AnimatePresence mode="popLayout" initial={false}>
				<motion.span
					key={value}
					className="ql-live-value"
					initial={{ opacity: 0, y: 8 }}
					animate={{ opacity: 1, y: 0 }}
					exit={{ opacity: 0, y: -8 }}
					transition={{ duration: 0.24, ease: [0.16, 1, 0.3, 1] }}
				>
					{value}
				</motion.span>
			</AnimatePresence>
		</span>
	);
}

interface QueueProgressTrackProps {
	ahead: number;
	totalWaiting: number;
}

/** Visual progress indicator showing how close the customer is to the front of the queue. */
export function QueueProgressTrack({
	ahead,
	totalWaiting,
}: QueueProgressTrackProps) {
	const safeTotal = Math.max(1, totalWaiting);
	const servedRatio = Math.max(0, Math.min(1, (safeTotal - ahead) / safeTotal));
	const percent = ahead === 0 ? 100 : Math.max(12, Math.round(servedRatio * 100));

	return (
		<div className="ql-progress-track-wrap" aria-hidden="true">
			<div className="ql-progress-track">
				<motion.div
					className={`ql-progress-fill ${ahead === 0 ? "is-next" : ""}`}
					initial={false}
					animate={{ width: `${percent}%` }}
					transition={{ type: "spring", stiffness: 220, damping: 28 }}
				/>
			</div>
			<div className="ql-progress-labels">
				<span>Joined</span>
				<span>{ahead === 0 ? "Up next" : `${percent}% there`}</span>
			</div>
		</div>
	);
}

/** Crisp geometric checkmark with stroke-draw animation for ticket confirmation. */
export function AnimatedCheckIcon({ size = 30 }: { size?: number }) {
	return (
		<svg
			width={size}
			height={size}
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			strokeWidth="2.5"
			strokeLinecap="round"
			strokeLinejoin="round"
			aria-hidden="true"
		>
			<motion.path
				d="M5 13l4 4L19 7"
				initial={{ pathLength: 0.15 }}
				animate={{ pathLength: 1 }}
				transition={{ duration: 0.38, ease: [0.16, 1, 0.3, 1] }}
			/>
		</svg>
	);
}
