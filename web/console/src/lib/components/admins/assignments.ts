import type { AdminAssignment, AdminAssignmentInput } from '#lib/api/index.js';

/** A role held for the whole panel rather than one application. */
export const PANEL = 'panel';

/** One role row in the administrator dialog: a role, and where it is held. `key` keeps rows
    apart while the role is still being chosen. */
export type Row = { key: number; role: string; scope: string };

export function rowsFrom(assignments: AdminAssignment[], next: () => number): Row[] {
	return assignments.map((assignment) => ({
		key: next(),
		role: assignment.role.id,
		scope: assignment.application?.id ?? PANEL
	}));
}

export function assignmentsOf(rows: Row[]): AdminAssignmentInput[] {
	return rows.map((row) => ({
		role_id: row.role,
		application_id: row.scope === PANEL ? null : row.scope
	}));
}

/** Changes a row's role. super_admin is only ever held for the whole panel, so choosing it moves
    the row there. */
export function withRole(rows: Row[], key: number, role: string, superRole?: string): Row[] {
	return rows.map((row) =>
		row.key === key ? { ...row, role, scope: role === superRole ? PANEL : row.scope } : row
	);
}
