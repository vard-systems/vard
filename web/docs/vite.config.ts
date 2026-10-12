import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';
import { loadEnv } from 'vite';
import { defineConfig } from 'vite';

// The docs are Markdown rendered while building: every page is prerendered,
// and the Node server adapter-node builds only serves those files — the same
// server and Dockerfile as the other two apps.
//
// BASE_PATH publishes them under a path, such as /docs, rather than at the
// root of a host. API_URL is a running server, whose discovery document the
// dev server passes through so a reader can compare the guides with it.
const { API_URL = 'http://localhost:8080', BASE_PATH = '' } = loadEnv(
	process.env.NODE_ENV ?? 'development',
	process.cwd(),
	''
);
if (BASE_PATH !== '' && !/^\/.*[^/]$/.test(BASE_PATH)) {
	throw new Error(`BASE_PATH is ${BASE_PATH}: it starts with a slash and does not end with one`);
}
const base = BASE_PATH as '' | `/${string}`;

export default defineConfig({
	// id 5173, console 5174, docs 5175, as scripts/start.sh expects.
	server: {
		port: 5175,
		strictPort: true,
		proxy: { '/.well-known': { target: API_URL, xfwd: true } }
	},

	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},
			adapter: adapter(),
			// Absolute, because a page is rendered once and its links must not depend
			// on which page asked first (see the cache in $lib/server/markdown.ts).
			paths: { base, relative: false },
			prerender: { handleHttpError: 'fail', handleMissingId: 'fail' }
		})
	]
});
