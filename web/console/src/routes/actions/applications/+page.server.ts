import { applicationsApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	create: action(applicationsApi.create),
	update: action(applicationsApi.update)
} satisfies Actions;
