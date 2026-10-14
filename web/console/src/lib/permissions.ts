import type { Admin, AdminPermissionName } from '#lib/api/index.js';

/**
 * What the signed-in administrator may do, for deciding what to show. The server checks every
 * request itself; hiding a control is only convenience.
 */

export function can(
	admin: Admin | null | undefined,
	permission: AdminPermissionName,
	application?: string
): boolean {
	if (!admin) return false;
	if (admin.is_super_admin || admin.permissions.includes(permission)) return true;

	return (
		application !== undefined && (admin.scoped_permissions[application] ?? []).includes(permission)
	);
}

export function canAnywhere(
	admin: Admin | null | undefined,
	permission: AdminPermissionName
): boolean {
	if (!admin) return false;
	if (can(admin, permission)) return true;

	return Object.values(admin.scoped_permissions).some((granted) => granted.includes(permission));
}
