<script lang="ts">
	import { resolve } from '$app/paths';
	import { organizationApi, type Admin, type OrganizationBrand } from '#lib/api/index.js';
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

	// The organization's app logo, or the panel's own until it uploads one.
	const logo = $derived(organizationApi.logoUrl(organization.logo_version) || BRAND.logo);

	// Only an organisation's own console needs to say what it runs on; the
	// product's own would say its name twice.
	const powered = $derived(name !== BRAND.name);
	const label = $derived(powered ? `${name} console, powered by ${BRAND.name}` : `${name} console`);
</script>

<header class:mini={shell.collapsed}>
	<div class="menu-control">
		<IconButton
			icon={shell.menuOpen ? 'ri-close-line' : 'ri-menu-line'}
			label={shell.menuOpen ? 'Close the menu' : 'Open the menu'}
			size="sm"
			placement="right"
			onclick={() => shell.setMenu(!shell.menuOpen)}
		/>
	</div>

	<div class="brand-column">
		<a class="brand" href={resolve('/(panel)')} aria-label={label}>
			<Thumb src={logo} text={initials(name)} size="sm" bare />

			<span class="title">
				<span class="wordmark" title={name}>{name}</span>
				{#if powered}
					<span class="divider" aria-hidden="true">/</span>
					<span class="product">{BRAND.name}</span>
				{/if}
			</span>
		</a>
	</div>

	<div class="fold-control">
		<IconButton
			icon={shell.collapsed ? 'ri-sidebar-unfold-line' : 'ri-sidebar-fold-line'}
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
		/* Centres the 34px logo in a folded, 56px sidebar, so it stays put. */
		padding: 0 var(--space-2) 0 11px;
		color: var(--color-text);
		text-decoration: none;
		white-space: nowrap;
	}

	.brand:focus-visible {
		outline: 2px solid var(--color-info);
		outline-offset: -4px;
	}

	/* The organization's name, then, smaller and quieter, the product it runs
	   on. A long name gives way; the product's stays whole. */
	.title {
		display: flex;
		align-items: baseline;
		gap: 6px;
		min-width: 0;
		line-height: 1;
		transition: opacity var(--speed);
	}

	/* Tall enough for every letter's ascender and descender: the box clips
	   whatever it does not hold, so a long name can end in an ellipsis. */
	.wordmark {
		overflow: hidden;
		padding-block: 2px;
		line-height: 1.3;
		text-overflow: ellipsis;
		color: var(--color-text);
		font-size: var(--text-xl);
		font-weight: 500;
		letter-spacing: -0.02em;
	}

	.divider,
	.product {
		flex-shrink: 0;
		font-size: 10px;
		font-weight: 500;
		letter-spacing: 0.12em;
		text-transform: uppercase;
	}

	.divider {
		color: var(--color-text-disabled);
	}

	.product {
		color: var(--color-text-hint);
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

		.mini .title {
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

	/* A phone's header has no room for the product beside the organization's name. */
	@media (max-width: 40rem) {
		.divider,
		.product {
			display: none;
		}
	}
</style>
