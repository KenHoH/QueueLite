export interface SSESubscription {
	url: string;
	event: string;
	onEvent: (rawData: string) => void;
	onVisible?: () => void;
}

/**
 * Manages a credentialed EventSource tied to page visibility (closes when hidden, reconnects when visible).
 * Returns a cleanup function that permanently closes the subscription.
 */
export function subscribeSSE({
	url,
	event,
	onEvent,
	onVisible,
}: SSESubscription): () => void {
	let source: EventSource | null = null;
	let closed = false;

	const disconnect = () => {
		if (source) {
			source.close();
			source = null;
		}
	};

	const connect = () => {
		if (closed || source || document.hidden || typeof EventSource === "undefined") {
			return;
		}
		const next = new EventSource(url, { withCredentials: true });
		source = next;
		next.addEventListener(event, (message: MessageEvent<string>) => {
			if (!closed) onEvent(message.data);
		});
		next.addEventListener("error", () => {
			if (next.readyState === EventSource.CLOSED) disconnect();
		});
	};

	const handleVisibility = () => {
		if (document.hidden) {
			disconnect();
		} else if (!closed) {
			onVisible?.();
			connect();
		}
	};

	connect();
	document.addEventListener("visibilitychange", handleVisibility);

	return () => {
		closed = true;
		disconnect();
		document.removeEventListener("visibilitychange", handleVisibility);
	};
}
