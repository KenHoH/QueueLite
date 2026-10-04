import { APIError, parseAPIError } from './errors';
import { validateAPIResponse } from './responseSchema';
export interface RequestOptions { signal?: AbortSignal }
interface APIRequestOptions extends RequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  body?: unknown;
  query?: object;
}
export function createAPIClient(baseURL = '/api', fetcher: typeof fetch = (...args) => fetch(...args), config: { timeoutMs?: number; validate?: typeof validateAPIResponse } = {}) {
  return async function request<T>(path: string, options: APIRequestOptions = {}): Promise<T> {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(options.query ?? {})) {
      if (value !== undefined && value !== null) query.set(key, String(value));
    }
    const suffix = query.size ? `?${query}` : '';
    let response: Response;
    let raw: string;
    const deadline = config.timeoutMs ? new AbortController() : undefined;
    const timer = deadline ? setTimeout(() => deadline.abort(), config.timeoutMs) : undefined;
    const signal = deadline ? (options.signal ? AbortSignal.any([options.signal, deadline.signal]) : deadline.signal) : options.signal;
    try {
      response = await fetcher(`${baseURL.replace(/\/$/, '')}${path}${suffix}`, {
        method: options.method ?? 'GET', credentials: 'include', signal,
        headers: { Accept: 'application/json', ...(options.body === undefined ? {} : { 'Content-Type': 'application/json' }) },
        body: options.body === undefined ? undefined : JSON.stringify(options.body),
      });
      raw = await response.text();
    } catch (cause) {
      if (options.signal?.aborted) throw cause;
      if (deadline?.signal.aborted) throw new APIError(0, 'CLIENT_TIMEOUT', 'QueueLite took too long to respond. Check the current state before trying again.', 'network', cause);
      if (cause instanceof Error && cause.name === 'AbortError') throw cause;
      throw new APIError(0, 'CLIENT_NETWORK_ERROR', 'Unable to reach QueueLite. Please try again.', 'network', cause);
    } finally { if (timer) clearTimeout(timer); }
    let body: unknown;
    try { body = raw ? JSON.parse(raw) : undefined; } catch { body = raw; }
    if (!response.ok) throw parseAPIError(response.status, body);
    if (response.status === 204) return (config.validate ? config.validate(path, options.method ?? 'GET', undefined, response.status) : undefined) as T;
    if (!raw || typeof body === 'string') {
      throw new APIError(response.status, 'CLIENT_INVALID_RESPONSE', 'QueueLite returned an unexpected response.', 'response');
    }
    return (config.validate ? config.validate(path, options.method ?? 'GET', body, response.status) : body) as T;
  };
}
export const api = createAPIClient(import.meta.env.VITE_API_BASE_URL || '/api', undefined, { timeoutMs: 20000, validate: validateAPIResponse });
export const pathID = (id: string) => encodeURIComponent(id);
