import { error, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';

import { ApiError, setupApi, type Admin, type AdminPermissionName } from '#lib/api/index.js';
import { can, canAnywhere } from '#lib/permissions.js';

const FORBIDDEN = 'Your roles do not allow you to see this page.';

/**
 * Awaits an API call made by a load, and turns its failure into what SvelteKit shows instead of
 * the page: the sign-in page when the session is gone, an error page with the API's own sentence
 * otherwise.
 */
export async function fromApi<T>(call: Promise<T>): Promise<T> {
	try {
		return await call;
	} catch (err) {
		if (!(err instanceof ApiError)) throw err;

		if (err.status === 401) redirect(307, resolve('/login'));
		if (err.status === 403) error(403, FORBIDDEN);
		if (err.status === 0) error(503, 'Could not reach the admin API.');
		error(err.status, err.message);
	}
}

export function requirePermission(
	admin: Admin,
	permission: AdminPermissionName | 'super_admin'
): void {
	const allowed = permission === 'super_admin' ? admin.is_super_admin : can(admin, permission);

	if (!allowed) error(403, FORBIDDEN);
}

export function requireAnywhere(admin: Admin, permission: AdminPermissionName): void {
	if (!canAnywhere(admin, permission)) error(403, FORBIDDEN);
}

/** Whether the installation has no administrator yet. Asked before anyone signs in, so a failure
 *  to answer is treated as "no": the sign-in page is the safe place to land. */
export async function setupRequired(fetch: typeof globalThis.fetch): Promise<boolean> {
	return setupApi
		.status(fetch)
		.then((answer) => answer.required)
		.catch(() => false);
}
