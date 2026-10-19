<script lang="ts">
	import { resolve } from '$app/paths';
	import { RiCloseLine, RiMenuLine, RiSidebarFoldLine, RiSidebarUnfoldLine } from '#lib/icons.js';
	import type { Admin, OrganizationBrand } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import { IconButton, ThemeToggle, Thumb } from '#lib/components/ui/index.js';
	import { useShell } from '#lib/state/shell.svelte.js';
	import { initials } from '#lib/utils/format.js';
	import AccountMenu from './AccountMenu.svelte';
	import CommandPalette from './CommandPalette.svelte';
	import HelpMenu from './HelpMenu.svelte';

	type Props = { admin: Admin; organization: OrganizationBrand };

	let { admin, organization }: Props = $props();

	const shell = useShell();

	const name = $derived(organization.name.trim() || BRAND.name);
</script>

<header class:mini={shell.collapsed}>
	<div class="menu-control">
		<IconButton
			icon={shell.menuOpen ? RiCloseLine : RiMenuLine}
			label={shell.menuOpen ? 'Close the menu' : 'Open the menu'}
			size="sm"
			placement="right"
			onclick={() => shell.setMenu(!shell.menuOpen)}
		/>
	</div>

	<div class="brand-column">
		<a class="brand" href={resolve('/(panel)')} aria-label="{name} Console">
			<Thumb
				src={organization.logo_url}
				text={initials(name)}
				size="xs"
				shape="circle"
				tone="accent"
				bare
			/>

			<span class="wordmark">Console</span>
		</a>
	</div>

	<div class="fold-control">
		<IconButton
			icon={shell.collapsed ? RiSidebarUnfoldLine : RiSidebarFoldLine}
			label={shell.collapsed ? 'Expand the sidebar' : 'Collapse the sidebar'}
			size="sm"
			onclick={shell.toggle}
		/>
	</div>

	<div class="end">
		<CommandPalette />
		<HelpMenu />
		<ThemeToggle size="sm" variant="ghost" />
		<span class="rule" aria-hidden="true"></span>
		<AccountMenu {admin} />
	</div>
</header>

<style>
	header {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		z-index: 10;
		display: flex;
		align-items: center;
		gap: var(--space-2);
		height: var(--header-height);
		padding-right: var(--page-gutter);
		border-bottom: 1px solid var(--color-border);
		background: var(--color-surface);
		overscroll-behavior: none;
	}

	.brand-column {
		display: flex;
		flex-shrink: 0;
		align-items: center;
		align-self: stretch;
		width: var(--sidebar-width);
		min-width: 0;
		overflow: hidden;
		border-right: 1px solid var(--color-border);
		background: var(--nav-surface);
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
		outline: 2px solid var(--color-info);
		outline-offset: -4px;
	}

	.wordmark {
		font-family: var(--font-display);
		font-size: var(--text-xl);
		font-weight: 600;
		line-height: 1;
		letter-spacing: -0.03em;
		transition: opacity var(--speed);
	}

	.end {
		display: flex;
		flex-shrink: 0;
		align-items: center;
		gap: 2px;
		margin-left: auto;
	}

	.rule {
		width: 1px;
		height: 20px;
		margin: 0 var(--space-2);
		background: var(--color-border);
	}

	@media (min-width: 55.0625rem) {
		.menu-control {
			display: none;
		}

		.mini .wordmark {
			opacity: 0;
		}
	}

	@media (max-width: 55rem) {
		header {
			gap: var(--space-1);
			padding-left: var(--space-2);
		}

		.fold-control {
			display: none;
		}

		.brand-column {
			width: auto;
			flex-shrink: 1;
			border-right-color: transparent;
			background: none;
		}

		.brand {
			gap: var(--space-2);
			padding: 0 var(--space-1);
		}

		.rule {
			margin: 0 var(--space-1);
		}
	}
</style>
