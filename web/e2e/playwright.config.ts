import { defineConfig } from '@playwright/test';

// Run by scripts/e2e.sh (make e2e), which raises the stack and names it here.
export default defineConfig({
	testDir: 'tests',
	// One stack and one database: tests that set it up run first, then the rest
	// one at a time, so none sees another's half-done state.
	workers: 1,
	fullyParallel: false,
	retries: 0,
	reporter: [['list']],
	use: {
		headless: true,
		trace: 'retain-on-failure'
	},
	projects: [
		{ name: 'setup', testMatch: /setup\.spec\.ts/ },
		{ name: 'flows', testIgnore: /setup\.spec\.ts/, dependencies: ['setup'] }
	]
});
