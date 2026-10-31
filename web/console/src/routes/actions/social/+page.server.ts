import { socialApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	create: action(socialApi.create),
	update: action(socialApi.update)
} satisfies Actions;
