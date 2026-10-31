import prettier from 'eslint-config-prettier';
import path from 'node:path';
import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import { defineConfig, includeIgnoreFile } from 'eslint/config';
import globals from 'globals';
import ts from 'typescript-eslint';

const gitignorePath = path.resolve(import.meta.dirname, '.gitignore');

// The type-aware rules build the whole TypeScript program for every file,
// which takes minutes; `make lint-types` (and CI) turns them on, `make check`
// leaves them to svelte-check's type-check.
const typeAware = process.env.LINT_TYPES === '1';

export default defineConfig(
	includeIgnoreFile(gitignorePath),
	js.configs.recommended,
	ts.configs.recommended,
	svelte.configs.recommended,
	prettier,
	svelte.configs.prettier,
	{
		languageOptions: { globals: { ...globals.browser, ...globals.node } },
		rules: {
			// typescript-eslint strongly recommend that you do not use the no-undef lint rule on TypeScript projects.
			// see: https://typescript-eslint.io/troubleshooting/faqs/eslint/#i-get-errors-from-the-no-undef-rule-about-global-variables-not-being-defined-even-though-there-are-no-typescript-errors
			'no-undef': 'off'
		}
	},
	...(typeAware
		? [
				{
					// The type-aware checks: what the compiler cannot say on its own.
					// A promise nobody waits on fails silently, and a deprecated API
					// is one a release will take away.
					files: ['**/*.ts', '**/*.svelte'],
					ignores: ['*.config.ts'],
					languageOptions: { parserOptions: { projectService: true } },
					rules: {
						'@typescript-eslint/no-deprecated': 'error',
						'@typescript-eslint/no-floating-promises': 'error',
						'@typescript-eslint/no-misused-promises': 'error'
					}
				}
			]
		: []),
	{
		files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: {
			parserOptions: {
				projectService: typeAware,
				extraFileExtensions: ['.svelte'],
				parser: ts.parser
			}
		}
	}
);
