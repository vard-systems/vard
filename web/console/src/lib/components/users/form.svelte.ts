import type { UserField, UserInput, UserRecord } from '#lib/api/index.js';

export type FieldValues = Record<string, string | boolean>;

/** The user dialog's fields: filled from a user, edited, sent back whole. Additional fields are
    edited as text and checkboxes and turned back into the API's shape by `toInput`. */
export class UserForm {
	email = $state('');
	isEmailVerified = $state(false);
	firstName = $state('');
	lastName = $state('');
	isActive = $state(true);
	password = $state('');
	confirmPassword = $state('');
	isPasswordTemporary = $state(true);
	values = $state<FieldValues>({});

	mismatch = $derived(this.confirmPassword !== '' && this.password !== this.confirmPassword);

	fill(record: UserRecord | null, extras: UserField[]): void {
		this.email = record?.email ?? '';
		this.isEmailVerified = record?.is_email_verified ?? false;
		this.firstName = record?.first_name ?? '';
		this.lastName = record?.last_name ?? '';
		this.isActive = record?.is_active ?? true;
		this.password = '';
		this.confirmPassword = '';
		// A new user's first password is one an administrator chose, so it is temporary.
		this.isPasswordTemporary = record ? record.is_password_temporary : true;

		this.values = Object.fromEntries(
			extras.map((field) => {
				const stored = record?.data?.[field.name];

				if (field.type === 'bool') return [field.name, stored === true];
				const text = typeof stored === 'string' || typeof stored === 'number' ? String(stored) : '';
				if (field.type === 'date') return [field.name, text.slice(0, 10)];

				return [field.name, text];
			})
		);
	}

	/** The body to save. `hasPassword` is whether the user will have one: only a password can be
	    temporary. Empty additional fields are left out rather than sent blank. */
	toInput(extras: UserField[], hasPassword: boolean): UserInput {
		const data: Record<string, unknown> = {};

		for (const field of extras) {
			const value = this.values[field.name];

			if (field.type === 'bool') {
				data[field.name] = value === true;
			} else if (String(value ?? '').trim() !== '') {
				data[field.name] = value;
			}
		}

		return {
			email: this.email.trim(),
			is_email_verified: this.isEmailVerified,
			first_name: this.firstName.trim(),
			last_name: this.lastName.trim(),
			is_active: this.isActive,
			is_password_temporary: this.isPasswordTemporary && hasPassword,
			...(this.password !== ''
				? { password: this.password, confirm_password: this.confirmPassword }
				: {}),
			data
		};
	}
}
