import type { RoleMapping } from '#lib/api/index.js';

/** The roles a user holds in one scope: global, or one application. */
export type ScopeRow = { key: string; name: string; clientId?: string; roles: RoleMapping[] };

/** Groups what a user holds by scope, global first as the API sends it, the roles given directly
    before those that come with them. Inherited roles are left out unless asked for. */
export function byScope(held: RoleMapping[], showInherited: boolean): ScopeRow[] {
	const rows: ScopeRow[] = [];

	for (const mapping of held) {
		if (!mapping.assigned && !showInherited) continue;

		const key = mapping.application?.id ?? 'global';
		let row = rows.find((it) => it.key === key);

		if (!row) {
			row = {
				key,
				name: mapping.application?.name ?? 'Global',
				clientId: mapping.application?.client_id,
				roles: []
			};
			rows.push(row);
		}

		row.roles.push(mapping);
	}

	for (const row of rows) {
		row.roles.sort((a, b) => Number(b.assigned) - Number(a.assigned));
	}

	return rows;
}

/** What a role's chip says on hover: what it is for, and how it relates to the others held. */
export function explain(role: RoleMapping, held: RoleMapping[]): string {
	const parts = [role.description];

	if (!role.assigned) {
		parts.push(`Comes with ${role.via.map((it) => it.name).join(', ')}`);
	} else {
		const gives = held.filter((it) => it.via.some((through) => through.id === role.id));
		if (gives.length > 0) {
			parts.push(
				`Also gives ${gives.map((it) => (it.application ? `${it.application.name} / ${it.name}` : it.name)).join(', ')}`
			);
		}
	}

	return parts.filter(Boolean).join(' · ') || role.name;
}
