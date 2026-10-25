/**
 * The console's theme: every colour, size, typeface and speed. `bun run theme` renders it to
 * lib/styles/tokens.css and fonts.css, and `bun run check` fails while they are behind, so
 * change values here only. Layout follows the PocketBase admin UI; colour is Telegram's, with
 * one brand blue (#0088cc).
 */
import type { Theme } from './types';

const systemSans = [
	'system-ui',
	'-apple-system',
	'BlinkMacSystemFont',
	"'Segoe UI'",
	"'Helvetica Neue'",
	"'Noto Sans'",
	'Ubuntu',
	'Cantarell',
	'Arial',
	'sans-serif',
	"'Apple Color Emoji'",
	"'Segoe UI Emoji'",
	"'Noto Color Emoji'"
].join(', ');

const systemMono = [
	'ui-monospace',
	'SFMono-Regular',
	"'SF Mono'",
	'Menlo',
	'Consolas',
	"'Liberation Mono'",
	"'DejaVu Sans Mono'",
	'monospace'
].join(', ');

/**
 * The light palette, which names every colour token. Values are solid, since transparency would
 * differ over each surface.
 */
const light = {
	'color-brand': '#0088cc',
	'color-brand-hover': '#0077b3',
	'color-brand-text': '#fff',
	'color-brand-subtle': '#e5f3fa',

	'color-surface': '#fff',
	'color-body': '#fff',
	'color-surface-alt': '#f7f7f8',
	'color-primary': 'var(--color-brand)',
	'color-primary-hover': 'var(--color-brand-hover)',
	'color-primary-text': 'var(--color-brand-text)',
	'color-secondary': '#f1f2f4',
	'color-secondary-alt': '#e6e8eb',
	'color-accent': 'var(--color-brand)',
	'color-accent-text': 'var(--color-brand-text)',

	'color-text': '#16191d',
	'color-text-hint': '#5b6168',
	'color-text-disabled': '#969ba1',

	'color-input': 'transparent',
	'color-input-border': '#d3d7dc',
	'color-input-border-hover': '#aab0b7',
	'color-border': '#e4e6e9',

	'color-success': '#31a24c',
	'color-danger': '#e53935',
	'color-warning': '#e8871e',
	'color-info': 'var(--color-brand)',

	'surface-success': '#e8f5eb',
	'surface-danger': '#fce9e9',
	'surface-info': 'var(--color-brand-subtle)',
	'surface-warning': '#fcf1e3',

	'shadow-sm': '0 1px 2px 0 rgb(0 0 0 / 8%)',
	'shadow-panel': '0 6px 16px -6px rgb(0 0 0 / 14%)',
	'shadow-md': '0 4px 20px 0 rgb(0 0 0 / 10%)',
	selection: '#b3dcf0',
	'color-overlay': 'rgb(0 0 0 / 30%)',

	'scrollbar-thumb': '#d4d6da',
	'scrollbar-thumb-hover': '#b6b9be',

	'tooltip-surface': '#2b2b2e',
	'tooltip-text': '#fff',

	'toast-surface': '#16191d',
	'toast-text': '#fff',
	'toast-hint': '#aab0b7',
	'toast-icon': '#3ea6e6',

	'row-hover': '#f7f7f8',

	'nav-surface': 'var(--color-surface-alt)',
	'nav-hover': 'var(--color-secondary-alt)',
	'nav-text': 'var(--color-text)',
	'nav-icon': 'var(--color-text-hint)',
	'nav-current': 'var(--color-brand)',
	'nav-current-text': 'var(--color-brand-text)',

	'popover-surface': 'var(--color-surface)',
	'popover-surface-alt': 'var(--color-surface-alt)',
	'popover-secondary': 'var(--color-secondary)',
	'popover-secondary-alt': 'var(--color-secondary-alt)',
	'popover-border': 'var(--color-border)',
	'popover-input-border': 'var(--color-input-border)',
	'popover-input-border-hover': 'var(--color-input-border-hover)',
	'popover-row-hover': 'var(--row-hover)',

	'dialog-surface': 'var(--popover-surface)',
	'dialog-surface-alt': 'var(--popover-surface-alt)',
	'dialog-secondary': 'var(--popover-secondary)',
	'dialog-secondary-alt': 'var(--popover-secondary-alt)',
	'dialog-border': 'var(--popover-border)',
	'dialog-input-border': 'var(--popover-input-border)',
	'dialog-input-border-hover': 'var(--popover-input-border-hover)',
	'dialog-row-hover': 'var(--popover-row-hover)',

	'syntax-foreground': 'var(--color-text)',
	'syntax-background': 'transparent',
	'syntax-token-keyword': '#0550ae',
	'syntax-token-string': '#0a7a3b',
	'syntax-token-string-expression': '#0a7a3b',
	'syntax-token-constant': '#b54708',
	'syntax-token-function': '#8250df',
	'syntax-token-parameter': '#953800',
	'syntax-token-comment': 'var(--color-text-hint)',
	'syntax-token-punctuation': 'var(--color-text-hint)',
	'syntax-token-link': 'var(--color-info)',

	'color-scheme': 'light'
} as const;

export type ColorName = keyof typeof light;

const black = '#000';

const dark: Partial<Record<ColorName, string>> = {
	'color-brand-hover': '#1a9ad9',
	'color-brand-subtle': '#06202e',

	'color-surface': black,
	'color-body': black,
	'color-surface-alt': '#0a0b0c',
	'color-secondary': '#16181c',
	'color-secondary-alt': '#202327',

	'color-text': '#e7e9ea',
	'color-text-hint': '#8b9096',
	'color-text-disabled': '#50555a',

	'color-input-border': '#2f3336',
	'color-input-border-hover': '#4c5257',
	'color-border': '#202327',

	'color-success': '#4cbb5e',
	'color-danger': '#ef5350',
	'color-warning': '#f0a040',
	'color-info': '#3ea6e6',

	'surface-success': '#081a0f',
	'surface-danger': '#220b0b',
	'surface-warning': '#221808',

	'shadow-sm': 'none',
	'shadow-panel': 'none',
	'shadow-md': '0 12px 32px -8px rgb(0 0 0 / 70%)',
	selection: '#0a4d73',
	'color-overlay': 'rgb(0 0 0 / 72%)',

	'scrollbar-thumb': '#2f3336',
	'scrollbar-thumb-hover': '#4c5257',

	'tooltip-surface': '#202327',

	'toast-surface': '#fff',
	'toast-text': '#16191d',
	'toast-hint': '#5b6168',
	'toast-icon': '#0088cc',

	'row-hover': '#0e1012',
	'nav-hover': 'var(--color-secondary)',

	'popover-surface': '#0f1114',
	'popover-surface-alt': '#15171a',
	'popover-secondary': '#1b1e22',
	'popover-secondary-alt': '#23262b',
	'popover-border': '#25282c',
	'popover-input-border': '#2f3336',
	'popover-input-border-hover': '#4c5257',
	'popover-row-hover': '#181b1f',

	'dialog-surface': '#0a0b0d',
	'dialog-surface-alt': '#101215',
	'dialog-secondary': '#16181c',
	'dialog-secondary-alt': '#1d2024',
	'dialog-border': '#1d2024',
	'dialog-input-border': '#2a2d31',
	'dialog-input-border-hover': '#43484d',
	'dialog-row-hover': '#111316',

	'syntax-token-keyword': '#79c0ff',
	'syntax-token-string': '#7ee787',
	'syntax-token-string-expression': '#7ee787',
	'syntax-token-constant': '#ffa657',
	'syntax-token-function': '#d2a8ff',
	'syntax-token-parameter': '#ffa657',

	'color-scheme': 'dark'
};

export const theme: Theme<ColorName> = {
	fonts: {
		// Two faces and no others: Chirp for every word, the organization's name in the header
		// included, and Google Sans Code for code. Chirp is bundled from assets/fonts: weights 400,
		// 500 and 700 (600 drawn from 700), Latin and Latin Extended. It is X's typeface, not an open font; see the README's Third-party assets
		// before shipping publicly.
		baseUrl: '../../assets/fonts/',
		subsets: {
			'latin-ext':
				'U+0100-02BA, U+02BD-02C5, U+02C7-02CC, U+02CE-02D7, U+02DD-02FF, U+0304, U+0308, U+0329, U+1D00-1DBF, U+1E00-1E9F, U+1EF2-1EFF, U+2020, U+20A0-20AB, U+20AD-20C0, U+2113, U+2C60-2C7F, U+A720-A7FF',
			latin:
				'U+0000-00FF, U+0131, U+0152-0153, U+02BB-02BC, U+02C6, U+02DA, U+02DC, U+0304, U+0308, U+0329, U+2000-206F, U+20AC, U+2122, U+2191, U+2193, U+2212, U+2215, U+FEFF, U+FFFD'
		},
		faces: [
			{
				family: 'Chirp',
				style: 'normal',
				weights: [400, 500, 700],
				subsets: ['latin-ext', 'latin'],
				file: (subset, weight) => `chirp/chirp-${subset}-${weight}-normal.woff2`
			},
			// Google Sans Code is the face for code: ids, keys, JSON. It is an
			// open font (SIL OFL 1.1, the licence beside the files), taken from
			// the @fontsource/google-sans-code release at the same weights and
			// alphabets as Chirp.
			{
				family: 'Google Sans Code',
				style: 'normal',
				weights: [400, 500, 700],
				subsets: ['latin-ext', 'latin'],
				file: (subset, weight) =>
					`google-sans-code/google-sans-code-${subset}-${weight}-normal.woff2`
			}
		]
	},

	scales: [
		{
			prefix: 'font',
			tokens: {
				'system-sans': systemSans,
				'system-mono': systemMono,
				sans: "'Chirp', var(--font-system-sans)",
				mono: "'Google Sans Code', var(--font-system-mono)"
			}
		},
		{
			prefix: 'text',
			tokens: { xs: '12px', sm: '13px', base: '14px', lg: '15px', xl: '19px', '2xl': '24px' }
		},
		{ prefix: '', tokens: { 'line-height': '1.5' } },
		{
			prefix: 'radius',
			tokens: {
				sm: '6px',
				md: '10px',
				lg: '14px',
				pill: '999px',
				field: 'var(--radius-sm)',
				control: 'var(--radius-pill)',
				surface: 'var(--radius-lg)'
			}
		},
		{
			prefix: 'space',
			tokens: { 1: '5px', 2: '10px', 3: '15px', 4: '20px', 5: '30px', 6: '40px' }
		},
		{
			prefix: '',
			tokens: {
				'control-height': '45px',
				'control-height-sm': '35px',
				'control-height-lg': '52px',
				'nav-item-height': '34px',
				'header-height': '55px',
				'sidebar-width': '240px',
				'content-max-width': '1400px',
				'form-max-width': '760px',
				'page-gutter': '16px',
				'dialog-column-width': '880px',
				'scrollbar-size': '10px'
			}
		},
		{ prefix: '', tokens: { 'speed-fast': '70ms', speed: '140ms', 'speed-slow': '200ms' } }
	],

	breakpoints: {
		phone: '34rem',
		small: '40rem',
		tablet: '48rem',
		sidebar: '55rem',
		laptop: '64rem',
		desktop: '80rem',
		wide: '90rem'
	},

	responsive: [
		{ minWidth: '48rem', tokens: { 'page-gutter': '24px' } },
		{ minWidth: '90rem', tokens: { 'page-gutter': '32px' } }
	],

	colors: { light, dark },

	layers: {
		swapped: {
			'color-surface': 'surface',
			'color-surface-alt': 'surface-alt',
			'color-secondary': 'secondary',
			'color-secondary-alt': 'secondary-alt',
			'color-border': 'border',
			'color-input-border': 'input-border',
			'color-input-border-hover': 'input-border-hover',
			'row-hover': 'row-hover'
		},
		sets: [
			{
				prefix: 'popover',
				selectors: [
					"[data-scope='menu'][data-part='content']",
					"[data-scope='select'][data-part='content']",
					"[data-scope='popover'][data-part='content']"
				]
			},
			{ prefix: 'dialog', selectors: ["[data-scope='dialog'][data-part='content']"] }
		]
	}
};
