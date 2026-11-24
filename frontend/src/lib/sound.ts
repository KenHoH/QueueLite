/**
 * Generates a gentle two-tone chime using the browser Web Audio API.
 * Used exclusively when the user opts into turn alerts on the live queue ticket page.
 */
export function playTurnChime(): void {
	if (typeof window === "undefined") return;
	const AudioCtx =
		window.AudioContext ||
		(window as unknown as { webkitAudioContext?: typeof AudioContext })
			.webkitAudioContext;
	if (!AudioCtx) return;

	try {
		const ctx = new AudioCtx();
		const now = ctx.currentTime;
		const notes = [
			{ freq: 587.33, start: 0, duration: 0.22 }, // D5
			{ freq: 880.0, start: 0.16, duration: 0.34 }, // A5
		];

		for (const note of notes) {
			const osc = ctx.createOscillator();
			const gain = ctx.createGain();
			osc.type = "sine";
			osc.frequency.setValueAtTime(note.freq, now + note.start);

			gain.gain.setValueAtTime(0.0001, now + note.start);
			gain.gain.exponentialRampToValueAtTime(0.14, now + note.start + 0.025);
			gain.gain.exponentialRampToValueAtTime(
				0.0001,
				now + note.start + note.duration,
			);

			osc.connect(gain);
			gain.connect(ctx.destination);
			osc.start(now + note.start);
			osc.stop(now + note.start + note.duration + 0.02);
		}

		window.setTimeout(() => {
			void ctx.close().catch(() => {});
		}, 900);
	} catch {
		// Ignore autoplay / audio hardware restrictions silently.
	}
}
