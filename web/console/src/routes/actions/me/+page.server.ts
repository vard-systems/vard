import { adminApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	changePassword: action(adminApi.changePassword),
	updateProfile: action(adminApi.updateProfile)
} satisfies Actions;
