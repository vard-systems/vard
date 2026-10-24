import { describe, expect, it } from 'vitest';
import type { RoleMapping } from '#lib/api/index.js';
import { byScope, explain } from './mappings';

const shop = { id: 'shop', name: 'Shop', client_id: 'shop-client' };

function role(name: string, change: Partial<RoleMapping> = {}): RoleMapping {
	return {
		id: name,
		name,
		description: '',
		application: null,
		composite: false,
		assigned: true,
		via: [],
		...change
	};
}

const held = [
	role('reader', { assigned: false, via: [{ id: 'editor', name: 'editor' }] }),
	role('editor', { composite: true, description: 'Writes posts' }),
	role('buyer', { application: shop })
];

describe('the roles a user holds', () => {
	it('are grouped by scope, given roles before inherited ones', () => {
		const rows = byScope(held, true);
		expect(rows.map((row) => [row.name, row.roles.map((it) => it.name)])).toEqual([
			['Global', ['editor', 'reader']],
			['Shop', ['buyer']]
		]);
		expect(rows[1].clientId).toBe('shop-client');
	});

	it('leave inherited roles out unless asked', () =>
		expect(byScope(held, false)[0].roles.map((it) => it.name)).toEqual(['editor']));

	it.each([
		['an inherited role says what it comes with', held[0], 'Comes with editor'],
		['a composite says what it gives', held[1], 'Writes posts · Also gives reader'],
		['a role with nothing to say is its name', held[2], 'buyer']
	])('%s', (_, mapping, expected) => expect(explain(mapping, held)).toBe(expected));
});
