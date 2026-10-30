import { usersApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	create: action(usersApi.create),
	update: action(usersApi.update),
	addField: action(usersApi.addField),
	updateField: action(usersApi.updateField)
} satisfies Actions;
