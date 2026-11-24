import {
	createContext,
	type ReactNode,
	useCallback,
	useContext,
	useMemo,
	useRef,
	useState,
} from "react";
import {
	playPop,
	playSoftError,
	playStepSuccess,
	playTap,
	playTicketIssued,
	playTurnChime,
} from "../lib/sound";

interface SoundContextValue {
	enabled: boolean;
	toggle: (preview?: "pop" | "chime") => boolean;
	setEnabled: (next: boolean) => void;
	tap: () => void;
	pop: () => void;
	ticketIssued: () => void;
	turnChime: () => void;
	stepSuccess: () => void;
	softError: () => void;
}

const noop = () => {};
const SoundContext = createContext<SoundContextValue>({
	enabled: false,
	toggle: () => false,
	setEnabled: noop,
	tap: noop,
	pop: noop,
	ticketIssued: noop,
	turnChime: noop,
	stepSuccess: noop,
	softError: noop,
});

export function useSound() {
	return useContext(SoundContext);
}

/**
 * In-memory opt-in sound preference provider.
 * Muted by default and intentionally avoids localStorage/sessionStorage.
 */
export function SoundProvider({ children }: { children: ReactNode }) {
	const [enabled, setEnabledState] = useState(false);
	const enabledRef = useRef(false);

	const setEnabled = useCallback((next: boolean) => {
		enabledRef.current = next;
		setEnabledState(next);
	}, []);

	const toggle = useCallback((preview: "pop" | "chime" = "pop") => {
		const next = !enabledRef.current;
		enabledRef.current = next;
		setEnabledState(next);
		if (next) {
			if (preview === "chime") playTurnChime();
			else playPop();
		}
		return next;
	}, []);

	const value = useMemo<SoundContextValue>(
		() => ({
			enabled,
			toggle,
			setEnabled,
			tap: () => {
				if (enabled) playTap();
			},
			pop: () => {
				if (enabled) playPop();
			},
			ticketIssued: () => {
				if (enabled) playTicketIssued();
			},
			turnChime: () => {
				if (enabled) playTurnChime();
			},
			stepSuccess: () => {
				if (enabled) playStepSuccess();
			},
			softError: () => {
				if (enabled) playSoftError();
			},
		}),
		[enabled, toggle],
	);

	return (
		<SoundContext.Provider value={value}>{children}</SoundContext.Provider>
	);
}
