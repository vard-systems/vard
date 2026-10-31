import { mfaApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	begin: action(mfaApi.begin),
	confirm: action(mfaApi.confirm)
} satisfies Actions;
