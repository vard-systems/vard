import { error } from '@sveltejs/kit';
import { ApiError, signIn } from '#lib/api/index.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	const token = url.searchParams.get('token') ?? '';
	const temporary = url.searchParams.get('reason') === 'temporary';

	if (!token) return { token, valid: false, temporary };

	try {
		const { valid } = await signIn.checkReset(token, fetch);
		return { token, valid, temporary };
	} catch (err) {
		if (err instanceof ApiError && err.status === 0) error(503, err.message);
		throw err;
	}
};
