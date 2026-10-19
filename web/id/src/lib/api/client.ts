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
}

export type Fetch = typeof globalThis.fetch;

type Options = {
	method?: 'GET' | 'POST' | 'PATCH' | 'DELETE';
	body?: unknown;
	fetch?: Fetch;
};

/** Calls the API at /api/v1 on this origin. From a load, pass the load's `fetch`. */
export async function request<T>(path: string, options: Options = {}): Promise<T> {
	const { method = 'GET', body, fetch: fetcher = globalThis.fetch } = options;

	const headers: Record<string, string> = {};
	if (body !== undefined) headers['Content-Type'] = 'application/json';

	let response: Response;
	try {
		response = await fetcher(`/api/v1${path}`, {
			method,
			credentials: 'same-origin',
			headers,
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	} catch {
		throw new ApiError(
			0,
			'Could not reach the server. Check your connection and try again.',
			'network'
		);
	}

	const payload: unknown = response.status === 204 ? {} : await response.json().catch(() => ({}));

	if (!response.ok) throw failureOf(response.status, payload);

	return payload as T;
}

/** The API's answer to a failed call, read only as far as it has the shape the API sends: a
 *  body from anything else in the way (a proxy's error page, a cut-off answer) gives the
 *  generic sentence rather than a broken one. */
function failureOf(status: number, payload: unknown): ApiError {
	const body = isRecord(payload) ? payload : {};
	const params = isRecord(body.params) ? body.params : {};

	return new ApiError(
		status,
		typeof body.error === 'string' ? body.error : 'Something went wrong. Try again.',
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

export function requiredSSO(err: unknown): { slug: string; name: string } | null {
	if (!(err instanceof ApiError) || err.code !== 'sso_required') return null;

	const { slug, name } = err.params;
	return typeof slug === 'string' && typeof name === 'string' ? { slug, name } : null;
}
