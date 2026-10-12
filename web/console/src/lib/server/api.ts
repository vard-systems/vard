import { error, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';

import type { Admin, AdminPermissionName } from '$lib/api';
import { can, canAnywhere } from '$lib/permissions';

async function call(path: string, fetch: typeof globalThis.fetch) {
	try {
		return await fetch(`/api/v1${path}`);
	} catch {
		error(503, 'Could not reach the admin API. Is it running, and is API_URL right?');
	}
}

export async function apiGet<T>(path: string, fetch: typeof globalThis.fetch): Promise<T> {
	const response = await call(path, fetch);

	if (response.status === 401) {
		redirect(307, resolve('/admin/login'));
	}

	if (response.status === 403) {
		error(403, 'Your roles do not allow you to see this page.');
	}

	if (!response.ok) {
		error(response.status, 'The server could not answer that');
	}

	return response.json() as Promise<T>;
}

export async function setupRequired(fetch: typeof globalThis.fetch): Promise<boolean> {
	const response = await call('/admin/setup', fetch);

	if (!response.ok) return false;

	const { required } = (await response.json()) as { required: boolean };

	return required;
}

export function requirePermission(
	admin: Admin,
	permission: AdminPermissionName | 'super_admin'
): void {
	const allowed = permission === 'super_admin' ? admin.is_super_admin : can(admin, permission);

	if (!allowed) {
		error(403, 'Your roles do not allow you to see this page.');
	}
}

export function requireAnywhere(admin: Admin, permission: AdminPermissionName): void {
	if (!canAnywhere(admin, permission)) {
		error(403, 'Your roles do not allow you to see this page.');
	}
}
