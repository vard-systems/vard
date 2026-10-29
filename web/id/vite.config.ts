import { fileURLToPath } from 'node:url';
import { defineConfig } from 'vitest/config';
import adapter from '@sveltejs/adapter-node';
import { sveltekit } from '@sveltejs/kit/vite';

export default defineConfig({
	// id 5173, console 5174. Under `make dev` the browser opens these: the dev
	// server renders the pages, and src/hooks.server.ts passes the API's paths
	// on to the API, as the built server does in production.
	//
	// The translations are read from i18n/ at the top of the repository,
	// which is outside this app, so the dev server is told it may serve that
	// directory. A build inlines the files and needs nothing.
	server: {
		port: 5173,
		strictPort: true,
		fs: { allow: [fileURLToPath(new URL('../../i18n', import.meta.url))] }
	},

	plugins: [
		sveltekit({
			compilerOptions: {
				// Force runes mode for the project, except for libraries. Can be removed in svelte 6.
				runes: ({ filename }) =>
					filename.split(/[/\\]/).includes('node_modules') ? undefined : true
			},

			// A Node server (`node build`) that renders each page before sending
			// it, title and all, and passes the API's paths on (hooks.server.ts).
			adapter: adapter(),

			// SvelteKit's own CSRF check refuses, in a built app, every form post
			// from another origin or from none, before hooks.server.ts runs. Those
			// are exactly what the provider takes: a backend redeeming a code, an
			// SPA calling the token endpoint, a SAML or Apple answer posted from
			// the provider's page. The API guards what a cookie can change
			// itself (internal/api/kit/csrf), and this app has no form action or
			// endpoint of its own for the check to protect; internal/api's
			// routing test holds it to that.
			csrf: { trustedOrigins: ['*'] },

			// Every script is the app's own: the ones SvelteKit writes into a
			// page carry a fresh nonce, the rest are files from 'self', so an
			// injected <script> does not run. Styles stay inline-able (Svelte's
			// transitions and component style attributes need it); images may be
			// logos and avatars on any https address an administrator gave. The
			// policy is sent as a header, so it can forbid framing too.
			csp: {
				mode: 'auto',
				directives: {
					'default-src': ['self'],
					'script-src': ['self'],
					'style-src': ['self', 'unsafe-inline'],
					'img-src': ['self', 'data:', 'https:'],
					'font-src': ['self'],
					'connect-src': ['self'],
					'object-src': ['none'],
					'base-uri': ['none'],
					'form-action': ['self'],
					'frame-ancestors': ['none']
				}
			}
		})
	],
	// Components mounted in a test need Svelte's browser build; the
	// accessibility specs mount them in jsdom.
	resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,
	test: {
		expect: { requireAssertions: true },
		environment: 'node',
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
