import { fileURLToPath } from 'node:url';
import { loadEnv } from 'vite';
import { defineConfig } from 'vitest/config';
import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';

// The API is reached on this app's own origin, as in production, where the
// reverse proxy routes these paths to it. Same origin means the session cookie
// belongs to this app's host — so the server-side renderer can read it — and
// the browser needs no CORS. API_URL is where the API really listens.
const { API_URL = 'http://localhost:8080' } = loadEnv(
	process.env.NODE_ENV ?? 'development',
	process.cwd(),
	''
);

const api = { target: API_URL, xfwd: true };
const proxy = {
	'/api/v1/account': api,
	'/oauth2': api,
	'/.well-known': api
};

export default defineConfig({
	// id 5173, console 5174 — in both modes of scripts/start.sh.
	//
	// The translations are read from i18n/ at the top of the repository,
	// which is outside this app, so the dev server is told it may serve that
	// directory. A build inlines the files and needs nothing.
	server: {
		port: 5173,
		strictPort: true,
		proxy,
		fs: { allow: [fileURLToPath(new URL('../../i18n', import.meta.url))] }
	},

	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// adapter-node builds a Node server, run behind the reverse proxy that
			// routes the API paths to the API (Caddy, in deploy/).
			adapter: adapter()
		})
	],
	test: {
		expect: { requireAssertions: true },
		projects: [
			{
				extends: './vite.config.ts',
				test: {
					name: 'server',
					environment: 'node',
					include: ['src/**/*.{test,spec}.{js,ts}'],
					exclude: ['src/**/*.svelte.{test,spec}.{js,ts}']
				}
			}
		]
	}
});
