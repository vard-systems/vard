<script lang="ts">
	import { RiCloseLine, RiGithubLine, RiMenuLine } from 'svelte-remixicon';
	import { BRAND } from '#lib/brand.js';
	import { href } from '#lib/docs/links.js';
	import Icon from '../Icon.svelte';
	import ThemeToggle from '../ThemeToggle.svelte';
	import ExportMenu from './ExportMenu.svelte';
	import Search from './Search.svelte';

	type Props = { menuOpen: boolean; onMenu: () => void };

	let { menuOpen, onMenu }: Props = $props();
</script>

<!-- The console's header: the logo block ruled off above the sidebar's
     column, then the bar: the Export menu for Postman, Bruno and OpenAPI,
     then the tools as quiet round icons — search, the source, the theme. -->
<header>
	<div class="brand-column">
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- href() is resolve() -->
		<a class="brand" href={href('/')} aria-label="{BRAND.name} Docs">
			<span class="mark" aria-hidden="true">{BRAND.name.slice(0, 1)}</span>
			<span class="names" aria-hidden="true">
				<strong>{BRAND.name}</strong>
				<small>Docs</small>
			</span>
		</a>
	</div>

	<div class="bar">
		<button
			type="button"
			class="control menu"
			aria-label={menuOpen ? 'Close the menu' : 'Open the menu'}
			aria-expanded={menuOpen}
			onclick={onMenu}
		>
			<Icon icon={menuOpen ? RiCloseLine : RiMenuLine} />
		</button>

		<div class="end">
			<ExportMenu />
			<Search />
			<!-- eslint-disable svelte/no-navigation-without-resolve -- href() is resolve(), and the source is another site -->
			<a
				class="control"
				href={BRAND.githubURL}
				target="_blank"
				rel="noopener noreferrer"
				data-tooltip="Source"
				aria-label="The source on GitHub"
			>
				<Icon icon={RiGithubLine} />
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
			<span class="rule" aria-hidden="true"></span>
			<ThemeToggle />
		</div>
	</div>
</header>

<style>
	header {
		position: fixed;
		top: 0;
		right: 0;
		left: 0;
		z-index: 10;
		display: flex;
		align-items: center;
		height: var(--header-height);
		border-bottom: 1px solid var(--color-border);
		background: var(--color-surface);
	}

	/* Exactly as wide as the sidebar, ruled off on the same line as its edge,
	   so the two read as one block. */
	.brand-column {
		display: flex;
		flex-shrink: 0;
		align-self: stretch;
		align-items: center;
		width: var(--sidebar-width);
		border-right: 1px solid var(--color-border);
	}

	.brand {
		display: flex;
		align-items: center;
		gap: 10px;
		min-width: 0;
		height: 100%;
		padding: 0 var(--space-3) 0 14px;
		color: var(--color-text);
		text-decoration: none;
		white-space: nowrap;
	}

	.brand:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: -4px;
	}

	/* The console's thumb: a round mark in the brand colour. */
	.mark {
		display: grid;
		place-items: center;
		width: 28px;
		height: 28px;
		border-radius: var(--radius-pill);
		background: var(--color-accent);
		color: var(--color-accent-text);
		font-size: var(--text-sm);
		font-weight: 700;
	}

	.names {
		display: flex;
		flex-direction: column;
		line-height: 1.25;
	}

	.names strong {
		font-size: var(--text-base);
		font-weight: 600;
	}

	.names small {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.bar {
		display: flex;
		flex: 1;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
		height: 100%;
		padding: 0 var(--page-gutter) 0 var(--space-2);
	}

	.menu {
		display: none;
	}

	.end {
		display: flex;
		align-items: center;
		gap: 4px;
		margin-left: auto;
	}

	.rule {
		width: 1px;
		height: 20px;
		margin: 0 var(--space-2);
		background: var(--color-border);
	}

	/* No column on a narrow screen: the logo block is only as wide as the
	   mark, and the control beside it opens the sidebar as a panel. */
	@media (max-width: 55rem) {
		.brand-column {
			width: auto;
			border-right-color: transparent;
		}

		.brand {
			padding: 0 var(--space-1) 0 var(--space-3);
		}

		.bar {
			padding-left: 0;
		}

		.menu {
			display: inline-flex;
		}
	}

	@media (max-width: 40rem) {
		.names small {
			display: none;
		}

		.rule {
			margin: 0 var(--space-1);
		}
	}
</style>
