import type { Admin, Application, Role, UserRoleRef } from '#lib/api/index.js';
import { can } from '#lib/permissions.js';

export function reachable(from: string[], roles: Role[]): Set<string> {
	const byId = new Map(roles.map((role) => [role.id, role]));
	const seen = new Set<string>();
	const stack = [...from];

	while (stack.length > 0) {
		const id = stack.pop()!;
		const role = byId.get(id);

		if (!role || seen.has(id)) continue;
		seen.add(id);

		for (const inherited of role.inherits) stack.push(inherited.id);
	}

	return seen;
}

export function wouldCycle(role: string | undefined, candidate: string, roles: Role[]): boolean {
	if (!role) return false;

	return candidate === role || reachable([candidate], roles).has(role);
}

export function mayInherit(scope: string | null, other: Pick<Role, 'application_id'>): boolean {
	return scope === null || other.application_id === null || other.application_id === scope;
}

export function scopeName(role: Pick<UserRoleRef, 'application_id'>, apps: Application[]): string {
	if (role.application_id === null) return 'Global';

	return apps.find((app) => app.id === role.application_id)?.name ?? 'Unknown application';
}

export function canAssign(admin: Admin, role: Pick<Role, 'application_id'>): boolean {
	return role.application_id === null
		? can(admin, 'role_assignments.write')
		: can(admin, 'role_assignments.write', role.application_id);
}

export function canEditRoles(admin: Admin, scope: string | null): boolean {
	return scope === null ? can(admin, 'user_roles.write') : can(admin, 'user_roles.write', scope);
}

export function byScope(
	roles: Role[],
	apps: Application[]
): { scope: string | null; name: string; roles: Role[] }[] {
	const groups = new Map<string | null, Role[]>();

	for (const role of roles) {
		const group = groups.get(role.application_id) ?? [];
		group.push(role);
		groups.set(role.application_id, group);
	}

	return [...groups.entries()]
		.map(([scope, group]) => ({
			scope,
			name: scopeName({ application_id: scope }, apps),
			roles: group.sort((a, b) => a.name.localeCompare(b.name))
		}))
		.sort((a, b) => (a.scope === null ? -1 : b.scope === null ? 1 : a.name.localeCompare(b.name)));
}

export function tidyName(value: string): string {
	return value.trim().toLowerCase();
}
