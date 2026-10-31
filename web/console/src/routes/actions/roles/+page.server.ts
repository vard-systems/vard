import { rolesApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	create: action(rolesApi.create),
	update: action(rolesApi.update)
} satisfies Actions;
