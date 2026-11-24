/**
 * Pure Web Audio API UI sound synthesizer for QueueLite.
 * Zero external audio assets, zero network calls, and strictly opt-in.
 */

let sharedCtx: AudioContext | null = null;

function getAudioContext(): AudioContext | null {
	if (typeof window === "undefined") return null;
	const AudioCtx =
		window.AudioContext ||
		(window as unknown as { webkitAudioContext?: typeof AudioContext })
			.webkitAudioContext;
	if (!AudioCtx) return null;

	try {
		if (!sharedCtx || sharedCtx.state === "closed") {
			sharedCtx = new AudioCtx();
		}
		if (sharedCtx.state === "suspended") {
			void sharedCtx.resume().catch(() => {});
		}
		return sharedCtx;
	} catch {
		return null;
	}
}

interface ToneSpec {
	freq: number;
	endFreq?: number;
	start: number;
	duration: number;
	gain: number;
	type?: OscillatorType;
}

function playSequence(tones: ToneSpec[]): void {
	const ctx = getAudioContext();
	if (!ctx) return;

	try {
		const base = ctx.currentTime;
		for (const tone of tones) {
			const osc = ctx.createOscillator();
			const amp = ctx.createGain();
			osc.type = tone.type ?? "sine";
			osc.frequency.setValueAtTime(tone.freq, base + tone.start);
			if (tone.endFreq !== undefined) {
				osc.frequency.exponentialRampToValueAtTime(
					tone.endFreq,
					base + tone.start + tone.duration,
				);
			}

			amp.gain.setValueAtTime(0.0001, base + tone.start);
			amp.gain.exponentialRampToValueAtTime(
				tone.gain,
				base + tone.start + Math.min(0.018, tone.duration * 0.35),
			);
			amp.gain.exponentialRampToValueAtTime(
				0.0001,
				base + tone.start + tone.duration,
			);

			osc.connect(amp);
			amp.connect(ctx.destination);
			osc.start(base + tone.start);
			osc.stop(base + tone.start + tone.duration + 0.02);
		}
	} catch {
		// Ignore autoplay or audio hardware restrictions silently.
	}
}

/** Subtle crisp tap for filter pills, tabs, and lightweight switches. */
export function playTap(): void {
	playSequence([
		{ freq: 520, endFreq: 420, start: 0, duration: 0.04, gain: 0.055, type: "sine" },
	]);
}

/** Warm pop for enabling sound, refreshing lists, or opening panels. */
export function playPop(): void {
	playSequence([
		{ freq: 420, endFreq: 660, start: 0, duration: 0.085, gain: 0.075, type: "sine" },
	]);
}

/** Ascending botanical arpeggio when a new queue ticket is issued. */
export function playTicketIssued(): void {
	playSequence([
		{ freq: 523.25, start: 0, duration: 0.16, gain: 0.09, type: "triangle" }, // C5
		{ freq: 659.25, start: 0.09, duration: 0.18, gain: 0.095, type: "triangle" }, // E5
		{ freq: 783.99, start: 0.18, duration: 0.32, gain: 0.11, type: "sine" }, // G5
	]);
}

/** Welcoming two-tone bell chime when a customer's turn is called. */
export function playTurnChime(): void {
	playSequence([
		{ freq: 587.33, start: 0, duration: 0.24, gain: 0.12, type: "sine" }, // D5
		{ freq: 880.0, start: 0.15, duration: 0.38, gain: 0.13, type: "sine" }, // A5
		{ freq: 1174.66, start: 0.15, duration: 0.28, gain: 0.04, type: "triangle" }, // soft shimmer overtone
	]);
}

/** Crisp 2-note cadence for copy confirmations and completing counter steps. */
export function playStepSuccess(): void {
	playSequence([
		{ freq: 659.25, start: 0, duration: 0.11, gain: 0.075, type: "sine" }, // E5
		{ freq: 987.77, start: 0.08, duration: 0.19, gain: 0.085, type: "sine" }, // B5
	]);
}

/** Gentle low double-pulse when a form fails validation. */
export function playSoftError(): void {
	playSequence([
		{ freq: 240, endFreq: 195, start: 0, duration: 0.09, gain: 0.08, type: "triangle" },
		{ freq: 195, endFreq: 165, start: 0.1, duration: 0.11, gain: 0.07, type: "triangle" },
	]);
}
