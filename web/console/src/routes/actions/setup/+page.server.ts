import { adminApi, setupApi } from '#lib/api/index.js';
import { act } from '#lib/server/actions.js';
import type { Actions } from './$types';

type SetupInput = Parameters<typeof setupApi.create>[0];

export const actions = {
	// The first administrator is made, then signed in, as one step.
	create: (event) =>
		act(event, async ([input]: [SetupInput]) => {
			await setupApi.create(input);
			return adminApi.login(input.email, input.password);
		})
} satisfies Actions;
