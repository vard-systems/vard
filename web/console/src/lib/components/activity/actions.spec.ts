import { describe as group, expect, it } from 'vitest';

import type { ActivityEvent } from '#lib/api/index.js';
import { describe } from './actions';

function event(changes: Partial<ActivityEvent>): ActivityEvent {
	return {
		id: '1',
		action: 'application.created',
		actor: 'root@example.com',
		ip: '127.0.0.1',
		target: null,
		created_at: '2026-09-14T10:00:00Z',
		...changes
	};
}

group('describe', () => {
	it('names the target when the server did', () => {
		const got = describe(event({ target: { type: 'application', id: 'a', name: 'Shop' } }));

		expect([got.actor, got.verb, got.subject, got.named]).toEqual([
			'root@example.com',
			'registered application',
			'Shop',
			true
		]);
		expect(got.tone).toBe('success');
	});

	it('says what kind of record it was when the name is hidden or gone', () => {
		const got = describe(event({ action: 'user.deleted', target: { type: 'user', id: 'u' } }));

		expect(got.subject).toBe('a user');
		expect(got.named).toBe(false);
	});

	it('does not repeat the account on a sign-in', () => {
		const got = describe(
			event({
				action: 'admin.login_failed',
				target: { type: 'admin_user', id: 'a', name: 'root@example.com' },
				detail: 'wrong password'
			})
		);

		expect(got.subject).toBe('');
		expect(got.detail).toBe('wrong password');
		expect(got.category).toBe('access');
	});

	it('describes a user signing in to an application as themselves', () => {
		const got = describe(
			event({
				action: 'user.login_failed',
				actor: 'ada@example.com',
				target: { type: 'user', id: 'u', name: 'ada@example.com' },
				detail: 'wrong password'
			})
		);

		expect([got.actor, got.subject, got.detail, got.tone]).toEqual([
			'ada@example.com',
			'',
			'wrong password',
			'danger'
		]);
	});

	it('names the API an application was given', () => {
		const got = describe(
			event({
				action: 'application.api_authorized',
				target: { type: 'application', id: 'a', name: 'Shop' },
				detail: 'https://api.example.com'
			})
		);

		expect(got.after).toBe('access to https://api.example.com');
		expect(got.detail).toBe('');
	});

	it('still shows an action it does not know', () => {
		const got = describe(event({ action: 'something.new', actor: '' }));

		expect([got.label, got.actor, got.category]).toEqual(['something.new', 'System', 'other']);
	});
});
