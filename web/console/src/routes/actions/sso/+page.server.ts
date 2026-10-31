import { ssoApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	create: action(ssoApi.create),
	update: action(ssoApi.update)
} satisfies Actions;
