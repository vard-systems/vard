import { describe, expect, it } from 'vitest';

import type { Role } from '#lib/api/index.js';
import { mayInherit, reachable, wouldCycle } from './roles';

function role(id: string, inherits: string[] = [], application_id: string | null = null): Role {
	return {
		id,
		name: id,
		application_id,
		inherits: inherits.map((it) => ({ id: it, name: it, application_id: null }))
	} as Role;
}

describe('reachable', () => {
	it('follows inheritance all the way down, and survives a loop', () => {
		const roles = [
			role('admin', ['editor']),
			role('editor', ['viewer']),
			role('viewer', ['admin'])
		];

		expect([...reachable(['editor'], roles)].sort()).toEqual(['admin', 'editor', 'viewer']);
	});
});

describe('wouldCycle', () => {
	const roles = [role('admin', ['editor']), role('editor', ['viewer']), role('viewer')];

	it('refuses a role including itself or a role above it', () => {
		expect(wouldCycle('viewer', 'viewer', roles)).toBe(true);
		expect(wouldCycle('viewer', 'admin', roles)).toBe(true);
	});

	it('allows including a role below', () => {
		expect(wouldCycle('admin', 'viewer', roles)).toBe(false);
	});
});

describe('mayInherit', () => {
	it('holds the scope rule the server holds', () => {
		expect(mayInherit(null, { application_id: 'shop' })).toBe(true);
		expect(mayInherit('shop', { application_id: null })).toBe(true);
		expect(mayInherit('shop', { application_id: 'shop' })).toBe(true);
		expect(mayInherit('shop', { application_id: 'blog' })).toBe(false);
	});
});
