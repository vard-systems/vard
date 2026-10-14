import { describe, expect, it } from 'vitest';

import type { Admin } from '#lib/api/index.js';
import { can, canAnywhere } from './permissions';

function admin(changes: Partial<Admin>): Admin {
	return {
		id: 'a',
		username: 'a@example.com',
		email: 'a@example.com',
		full_name: 'A',
		first_name: 'A',
		last_name: '',
		avatar_url: '',
		status: 'active',
		roles: [],
		permissions: [],
		scoped_permissions: {},
		is_super_admin: false,
		mfa_enabled: false,
		...changes
	};
}

describe('can', () => {
	const manager = admin({ scoped_permissions: { shop: ['applications.read'] } });

	it('allows a whole-panel permission everywhere', () => {
		const reader = admin({ permissions: ['applications.read'] });

		expect(can(reader, 'applications.read')).toBe(true);
		expect(can(reader, 'applications.read', 'shop')).toBe(true);
	});

	it('allows a scoped permission only for its application', () => {
		expect(can(manager, 'applications.read', 'shop')).toBe(true);
		expect(can(manager, 'applications.read', 'blog')).toBe(false);
		expect(can(manager, 'applications.read')).toBe(false);
	});

	it('allows a super admin anything, and nobody nothing', () => {
		expect(can(admin({ is_super_admin: true }), 'apis.write')).toBe(true);
		expect(can(null, 'users.read')).toBe(false);
	});
});

describe('canAnywhere', () => {
	it('is enough to hold the permission for one application', () => {
		const manager = admin({ scoped_permissions: { shop: ['user_roles.write'] } });

		expect(canAnywhere(manager, 'user_roles.write')).toBe(true);
		expect(canAnywhere(manager, 'users.read')).toBe(false);
	});
});
