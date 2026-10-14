import { defineEnvVars } from '@sveltejs/kit/env';

// API_URL is where the admin API really listens; development runs it on this machine.
export const variables = defineEnvVars({
	API_URL: { schema: (input) => input || 'http://localhost:8081' }
});
