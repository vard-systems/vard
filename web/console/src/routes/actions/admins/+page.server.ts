import { adminsApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	create: action(adminsApi.create),
	createRole: action(adminsApi.createRole),
	update: action(adminsApi.update),
	updateRole: action(adminsApi.updateRole)
} satisfies Actions;
