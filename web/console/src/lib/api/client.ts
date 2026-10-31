export class ApiError extends Error {
	constructor(
		readonly status: number,
		message: string,
		readonly code = 'unknown',
		readonly params: Record<string, string | number> = {}
	) {
		super(message);
		this.name = 'ApiError';
	}

	get isUnauthorized(): boolean {
		return this.status === 401;
	}
}

/** The fetch SvelteKit hands a load function. Passing it in is what makes `invalidate` work. */
export type Fetch = typeof globalThis.fetch;

type RequestOptions = {
	method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
	body?: unknown;
	fetch?: Fetch;
};

/** On the server, the fetch of the request being answered; see `useRequestFetch`. */
let requestFetch: (() => Fetch | undefined) | undefined;

/**
 * Has calls made on the server go through `get()`: a form action's call to the admin API then
 * passes through this app's own hook, with the caller's cookies, as the browser's would. Set
 * once, by hooks.server.ts.
 */
export function useRequestFetch(get: () => Fetch | undefined) {
	requestFetch = get;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
	const { method = 'GET', body, fetch: fetcher = requestFetch?.() ?? globalThis.fetch } = options;

	let response: Response;
	try {
		// The admin API is on this app's own origin: this app's server passes
		// /api/v1/admin on to it (src/hooks.server.ts).
		response = await fetcher(apiPath(path), {
			method,
			credentials: 'same-origin',
			headers: { 'Content-Type': 'application/json' },
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	} catch {
		throw new ApiError(0, 'Could not reach the server. Is the admin API running?', 'network');
	}

	const payload: unknown = response.status === 204 ? {} : await response.json().catch(() => ({}));

	if (!response.ok) throw failureOf(response.status, payload);

	return payload as T;
}

/** The API's answer to a failed call, read only as far as it has the shape the API sends: a
 *  body from anything else in the way (a proxy's error page, a cut-off answer) gives the
 *  generic sentence rather than a broken one. */
export function failureOf(status: number, payload: unknown): ApiError {
	const body = isRecord(payload) ? payload : {};
	const params = isRecord(body.params) ? body.params : {};

	return new ApiError(
		status,
		typeof body.error === 'string' ? body.error : 'Something went wrong',
		typeof body.code === 'string' ? body.code : 'unknown',
		Object.fromEntries(
			Object.entries(params).filter(
				(entry): entry is [string, string | number] =>
					typeof entry[1] === 'string' || typeof entry[1] === 'number'
			)
		)
	);
}

function isRecord(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/** An admin API path on this app's own origin, for what is not fetched as JSON (an image's src). */
export function apiPath(path: string): string {
	return `/api/v1${path}`;
}

export const api = {
	get: <T>(path: string, fetcher?: Fetch) => request<T>(path, { fetch: fetcher }),
	post: <T>(path: string, body?: unknown, fetcher?: Fetch) =>
		request<T>(path, { method: 'POST', body, fetch: fetcher }),
	put: <T>(path: string, body?: unknown, fetcher?: Fetch) =>
		request<T>(path, { method: 'PUT', body, fetch: fetcher }),
	patch: <T>(path: string, body?: unknown, fetcher?: Fetch) =>
		request<T>(path, { method: 'PATCH', body, fetch: fetcher }),
	delete: <T>(path: string, body?: unknown, fetcher?: Fetch) =>
		request<T>(path, { method: 'DELETE', body, fetch: fetcher })
};
