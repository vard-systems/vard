import { organizationApi, type OrganizationInput } from '#lib/api/index.js';
import { act } from '#lib/server/actions.js';
import type { Actions } from './$types';

export const actions = {
	update: (event) => act(event, (input: OrganizationInput) => organizationApi.update(input))
} satisfies Actions;
