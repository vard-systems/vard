import { describe, expect, it } from 'vitest';
import { PANEL, assignmentsOf, rowsFrom, withRole } from './assignments';

describe("an administrator's role rows", () => {
	let key = 0;
	const next = () => key++;

	it('come from the assignments, whole-panel ones under the panel', () =>
		expect(
			rowsFrom(
				[
					{ id: 'a', role: { id: 'auditor', name: 'auditor' }, application: null },
					{
						id: 'b',
						role: { id: 'editor', name: 'editor' },
						application: { id: 'shop', name: 'Shop' }
					}
				],
				next
			).map(({ role, scope }) => [role, scope])
		).toEqual([
			['auditor', PANEL],
			['editor', 'shop']
		]));

	it('go back as assignments, the panel as no application', () =>
		expect(
			assignmentsOf([
				{ key: 1, role: 'auditor', scope: PANEL },
				{ key: 2, role: 'editor', scope: 'shop' }
			])
		).toEqual([
			{ role_id: 'auditor', application_id: null },
			{ role_id: 'editor', application_id: 'shop' }
		]));

	it('move to the whole panel when super_admin is chosen', () =>
		expect(withRole([{ key: 1, role: '', scope: 'shop' }], 1, 'super', 'super')[0].scope).toBe(
			PANEL
		));

	it('keep their scope for any other role', () =>
		expect(withRole([{ key: 1, role: '', scope: 'shop' }], 1, 'editor', 'super')[0].scope).toBe(
			'shop'
		));
});
