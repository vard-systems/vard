import { apisApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	create: action(apisApi.create),
	update: action(apisApi.update)
} satisfies Actions;
