import type { UserField, UserRecord } from '#lib/api/index.js';

export function valueOf(user: UserRecord, field: UserField): unknown {
	if (field.is_builtin) {
		return (user as unknown as Record<string, unknown>)[field.name];
	}

	return user.data?.[field.name];
}

export function builtins(fields: UserField[]): UserField[] {
	return fields.filter((field) => field.is_builtin);
}

export function additional(fields: UserField[]): UserField[] {
	return fields.filter((field) => !field.is_builtin);
}
