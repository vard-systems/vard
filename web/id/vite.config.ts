import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';
import adapter from '@sveltejs/adapter-static';
import { sveltekit } from '@sveltejs/kit/vite';

// The paths the API answers. The browser calls them on this app's own origin;
// the dev server passes them on to the API (API_URL, set by scripts/start.sh),
// so cookies stay host-only and nothing is cross-origin. A built app needs
// none of this: the binary serves it on the API's own listener.
const apiPaths = ['/oauth2', '/.well-known', '/api/v1/account'];
const api = process.env.API_URL ?? 'http://localhost:8080';

export default defineConfig({
	// id 5173, console 5174.
	//
	// The translations are read from i18n/ at the top of the repository,
	// which is outside this app, so the dev server is told it may serve that
	// directory. A build inlines the files and needs nothing.
	server: {
		port: 5173,
		strictPort: true,
		proxy: Object.fromEntries(apiPaths.map((path) => [path, api])),
		fs: { allow: [fileURLToPath(new URL('../../i18n', import.meta.url))] }
	},

	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// A single-page app: build/ holds static files and one index.html for
			// every route, which the Go server embeds and serves (web/embed.go,
			// internal/webui).
			adapter: adapter({
				pages: 'build',
				assets: 'build',
				fallback: 'index.html',
				precompress: false
			}),

			// Every script is the app's own: the one SvelteKit writes into
			// index.html is allowed by its hash, in a <meta> tag, since a static
			// file cannot carry a fresh nonce; the rest are files from 'self', so
			// an injected <script> does not run. Styles stay inline-able
			// (Svelte's transitions and component style attributes need it);
			// images may be logos and avatars on any https address an
			// administrator gave. Framing is refused by internal/webui's header:
			// a <meta> cannot say frame-ancestors.
			csp: {
				mode: 'hash',
				directives: {
					'default-src': ['self'],
					'script-src': ['self'],
					'style-src': ['self', 'unsafe-inline'],
					'img-src': ['self', 'data:', 'https:'],
					'font-src': ['self'],
					'connect-src': ['self'],
					'object-src': ['none'],
					'base-uri': ['self']
				}
			}
		})
	],
	test: {
		expect: { requireAssertions: true },
		environment: 'node',
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
