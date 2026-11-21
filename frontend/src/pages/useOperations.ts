import { useCallback, useEffect, useRef, useState } from "react";
import { APIError } from "../api/errors";
import { subscribeSSE } from "../api/sse";
import { useAppState } from "../state/AppState";

/** One cancellable read at a time. Live updates arrive via SSE when eventSourceURL is supplied (no polling). */
export function useOperations<T>(
	load: (signal: AbortSignal) => Promise<T>,
	eventSourceURL?: string | false,
) {
	const { refreshUser } = useAppState();
	const [state, setState] = useState<{
		data: T | null;
		loading: boolean;
		error: unknown;
	}>({ data: null, loading: true, error: null });
	const [refresh, setRefresh] = useState(0);
	const controller = useRef<AbortController | null>(null);
	const retry = useCallback(() => {
		controller.current?.abort();
		setState({ data: null, loading: true, error: null });
		setRefresh((value) => value + 1);
	}, []);

	useEffect(() => {
		let disposed = false;
		let unsubscribe: (() => void) | null = null;

		async function read(): Promise<boolean> {
			controller.current?.abort();
			const request = new AbortController();
			controller.current = request;
			try {
				const data = await load(request.signal);
				if (!disposed && !request.signal.aborted)
					setState({ data, loading: false, error: null });
				return true;
			} catch (error) {
				if (!disposed && !request.signal.aborted) {
					unsubscribe?.();
					unsubscribe = null;
					setState({ data: null, loading: false, error });
					if (error instanceof APIError && error.status === 401)
						void refreshUser(true).catch(() => {});
				}
				return false;
			}
		}

		setState({ data: null, loading: true, error: null });
		void read().then((ok) => {
			if (disposed || !ok || typeof eventSourceURL !== "string" || !eventSourceURL)
				return;
			unsubscribe = subscribeSSE({
				url: eventSourceURL,
				event: "queue.update",
				onEvent: () => void read(),
				onVisible: () => void read(),
			});
		});

		return () => {
			disposed = true;
			unsubscribe?.();
			controller.current?.abort();
		};
	}, [load, refresh, eventSourceURL, refreshUser]);

	return { ...state, retry };
}
