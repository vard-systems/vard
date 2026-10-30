import { adminApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

// Signing in is the API's: these pass its answer on, and the session cookie it
// sets reaches the browser with the action's own answer.
export const actions = {
	login: action(adminApi.login),
	verifyMfa: action(adminApi.verifyMfa)
} satisfies Actions;
