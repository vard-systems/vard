import { cacheApi } from '#lib/api/index.js';
import { action } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = { update: action(cacheApi.update) } satisfies Actions;
