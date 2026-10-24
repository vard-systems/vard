import { ApiError, account, type ConsentPage } from '#lib/api/index.js';
import { requireUser } from '#lib/session.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	// Signed in first: the device is allowed for whoever this is.
	const user = await requireUser(fetch, url);
	const code = url.searchParams.get('user_code') ?? '';

	let device: ConsentPage | null = null;
	let problem: ApiError | null = null;
	if (code) {
		try {
			device = (await account.device(code, fetch)).device;
		} catch (err) {
			if (!(err instanceof ApiError)) throw err;
			problem = err;
		}
	}

	return {
		user,
		code,
		device,
		problem: problem ? { code: problem.code, status: problem.status } : null
	};
};
