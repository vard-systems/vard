import { describe, expect, it } from 'vitest';
import type { FieldType, UserField, UserRecord } from '#lib/api/index.js';
import { UserForm } from './form.svelte';

function field(name: string, type: FieldType): UserField {
	return {
		name,
		label: name,
		type,
		position: 0,
		is_builtin: false,
		is_required: false,
		is_unique: false,
		min: null,
		max: null,
		starts_with: ''
	};
}

const extras = [field('nickname', 'text'), field('birthday', 'date'), field('vip', 'bool')];

const ada = {
	id: '1',
	email: 'ada@example.com',
	is_email_verified: true,
	first_name: 'Ada',
	last_name: 'Lovelace',
	is_active: true,
	is_password_temporary: false,
	has_password: true,
	two_step_enabled: false,
	last_login_at: null,
	locked_until: null,
	agreements_accepted_at: null,
	roles: [],
	social_accounts: [],
	data: { nickname: 'Countess', birthday: '1815-12-10T00:00:00Z', vip: true },
	created_at: '',
	updated_at: ''
} satisfies UserRecord;

describe('UserForm', () => {
	it('fills additional fields as the inputs edit them', () => {
		const form = new UserForm();
		form.fill(ada, extras);

		expect(form.values).toEqual({ nickname: 'Countess', birthday: '1815-12-10', vip: true });
	});

	it('starts a new user with a temporary password', () => {
		const form = new UserForm();
		form.fill(null, extras);

		expect(form.isPasswordTemporary).toBe(true);
		expect(form.values).toEqual({ nickname: '', birthday: '', vip: false });
	});

	it('sends trimmed fields, leaves out empty ones, and no password unless one was typed', () => {
		const form = new UserForm();
		form.fill(ada, extras);
		form.email = ' ada@example.com ';
		form.values.nickname = '  ';

		const input = form.toInput(extras, true);
		expect(input).toMatchObject({
			email: 'ada@example.com',
			data: { birthday: '1815-12-10', vip: true }
		});
		expect(input).not.toHaveProperty('password');
		expect(input.data).not.toHaveProperty('nickname');
	});

	it('never marks a password temporary when there is none', () => {
		const form = new UserForm();
		form.fill(null, []);

		expect(form.toInput([], false).is_password_temporary).toBe(false);
	});

	it('notices two passwords that differ', () => {
		const form = new UserForm();
		form.password = 'one-password';
		form.confirmPassword = 'another';

		expect(form.mismatch).toBe(true);
	});
});
