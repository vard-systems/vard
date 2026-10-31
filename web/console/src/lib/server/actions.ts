import { error, fail, type RequestEvent } from '@sveltejs/kit';

import { ApiError } from '#lib/api/client.js';

/** What a refused action answers: the API's own problem, for the form to show. */
export type ActionProblem = {
	error: { status: number; message: string; code: string; params: Record<string, string | number> };
};

/**
 * Runs a form action: reads the form's `input` (what the page holds, as JSON; see
 * `#lib/forms.js`) from a form posted on this origin only, passes it to `call`, which asks the admin API through the typed client, and
 * answers with what the API answered. A refusal is answered as the API gave it, with its status.
 */
export async function act<I, T>(event: RequestEvent, call: (input: I) => Promise<T>) {
	// The call is made as this origin (hooks.server.ts), so the form has to
	// come from it. SvelteKit checks this too, but not under its dev server.
	if (event.request.headers.get('origin') !== event.url.origin) {
		error(403, 'Cross-site form submissions are forbidden');
	}

	const form = await event.request.formData();

	let input: I;
	try {
		input = JSON.parse(String(form.get('input') ?? 'null')) as I;
	} catch {
		return fail(400, problem(new ApiError(400, 'The form could not be read.', 'invalid_body')));
	}

	try {
		return await call(input);
	} catch (error) {
		if (error instanceof ApiError) return fail(error.status || 502, problem(error));
		throw error;
	}
}

function problem(error: ApiError): ActionProblem {
	return {
		error: { status: error.status, message: error.message, code: error.code, params: error.params }
	};
}

/**
 * A form action that calls one method of the typed client (`#lib/api/admin.js`), with the
 * arguments the form sends as its input, in order: `update: action(usersApi.update)` answers a
 * form whose input is `[id, changes]`. The path and the answer stay written once, in the client.
 */
export function action<F extends (...args: never[]) => Promise<unknown>>(call: F) {
	return (event: RequestEvent) => act(event, (args: Parameters<F>) => call(...args));
}
