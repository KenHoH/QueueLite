import { CheckCircle2, Info, X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useEffect,
	useRef,
	useState,
} from "react";

type ToastTone = "success" | "info";

interface ToastItem {
	id: number;
	message: string;
	tone: ToastTone;
}

interface ToastContextValue {
	show: (message: string, tone?: ToastTone) => void;
	success: (message: string) => void;
	info: (message: string) => void;
}

const noop = () => {};
const ToastContext = createContext<ToastContextValue>({
	show: noop,
	success: noop,
	info: noop,
});

export function useToast() {
	return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: ReactNode }) {
	const [toasts, setToasts] = useState<ToastItem[]>([]);
	const nextId = useRef(1);
	const timers = useRef<Map<number, number>>(new Map());

	const dismiss = useCallback((id: number) => {
		const timer = timers.current.get(id);
		if (timer !== undefined) {
			window.clearTimeout(timer);
			timers.current.delete(id);
		}
		setToasts((prev) => prev.filter((item) => item.id !== id));
	}, []);

	const show = useCallback(
		(message: string, tone: ToastTone = "info") => {
			const id = nextId.current++;
			setToasts((prev) => [...prev.slice(-2), { id, message, tone }]);
			const timer = window.setTimeout(() => dismiss(id), 3200);
			timers.current.set(id, timer);
		},
		[dismiss],
	);

	const success = useCallback(
		(message: string) => show(message, "success"),
		[show],
	);
	const info = useCallback((message: string) => show(message, "info"), [show]);

	useEffect(() => {
		const activeTimers = timers.current;
		return () => {
			for (const timer of activeTimers.values()) window.clearTimeout(timer);
			activeTimers.clear();
		};
	}, []);

	return (
		<ToastContext.Provider value={{ show, success, info }}>
			{children}
			<div
				className="ql-toast-viewport"
				role="region"
				aria-label="Notifications"
				aria-live="polite"
			>
				<AnimatePresence initial={false}>
					{toasts.map((item) => (
						<motion.div
							key={item.id}
							layout
							initial={{ opacity: 0, y: 14, scale: 0.96 }}
							animate={{ opacity: 1, y: 0, scale: 1 }}
							exit={{ opacity: 0, y: 8, scale: 0.96 }}
							transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
							className={`ql-toast ql-toast-${item.tone}`}
							role="status"
						>
							<span className="ql-toast-icon" aria-hidden="true">
								{item.tone === "success" ? (
									<CheckCircle2 size={18} />
								) : (
									<Info size={18} />
								)}
							</span>
							<span className="ql-toast-message">{item.message}</span>
							<button
								type="button"
								className="ql-toast-close"
								aria-label="Dismiss notification"
								onClick={() => dismiss(item.id)}
							>
								<X size={15} aria-hidden="true" />
							</button>
						</motion.div>
					))}
				</AnimatePresence>
			</div>
		</ToastContext.Provider>
	);
}
