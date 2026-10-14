<!--
	@component
	Downloads each API as a Postman collection (Bruno imports it too) or an OpenAPI document, all
	written by `make docs`.
-->
<script lang="ts">
	import { RiArrowDownSLine, RiBracesLine, RiDownload2Line, RiStackLine } from 'svelte-remixicon';
	import { BRAND } from '#lib/brand.js';
	import { href } from '#lib/docs/links.js';
	import Icon from '../Icon.svelte';

	let open = $state(false);
	let root = $state<HTMLElement>();

	const groups = [
		{
			label: 'Postman / Bruno collection',
			icon: RiStackLine,
			items: [
				{
					label: 'Public API',
					path: '/collections/public.postman_collection.json',
					file: `${BRAND.slug}-public-api.postman_collection.json`
				},
				{
					label: 'Admin API',
					path: '/collections/admin.postman_collection.json',
					file: `${BRAND.slug}-admin-api.postman_collection.json`
				}
			]
		},
		{
			label: 'OpenAPI 3.1',
			icon: RiBracesLine,
			items: [
				{
					label: 'Public API',
					path: '/openapi/public.json',
					file: `${BRAND.slug}-public-api.openapi.json`
				},
				{
					label: 'Admin API',
					path: '/openapi/admin.json',
					file: `${BRAND.slug}-admin-api.openapi.json`
				}
			]
		}
	];

	// Closed by a click anywhere else, and by Escape, like the console's menus.
	function onwindowclick(event: MouseEvent) {
		if (open && root && !root.contains(event.target as Node)) open = false;
	}
</script>

<svelte:window
	onclick={onwindowclick}
	onkeydown={(event) => event.key === 'Escape' && (open = false)}
/>

<div class="export" bind:this={root}>
	<button
		type="button"
		class="trigger"
		aria-haspopup="menu"
		aria-expanded={open}
		onclick={() => (open = !open)}
	>
		<Icon icon={RiDownload2Line} />
		<span class="label">Export</span>
		<Icon icon={RiArrowDownSLine} size="0.875rem" />
	</button>

	{#if open}
		<div class="menu" role="menu">
			{#each groups as group (group.label)}
				<p class="group">
					<Icon icon={group.icon} size="0.875rem" />
					{group.label}
				</p>
				{#each group.items as item (item.path)}
					<!-- eslint-disable svelte/no-navigation-without-resolve -- href() is resolve() -->
					<a
						role="menuitem"
						href={href(item.path)}
						download={item.file}
						onclick={() => (open = false)}
					>
						{item.label}
						<span>.json</span>
					</a>
					<!-- eslint-enable svelte/no-navigation-without-resolve -->
				{/each}
			{/each}
			<p class="hint">
				In Postman or Bruno: <strong>Import</strong>, then choose the file. Fill in the collection's
				variables — the addresses, and a secret or token.
			</p>
		</div>
	{/if}
</div>

<style>
	.export {
		position: relative;
	}

	/* A quiet pill: the one worded control in the header, since "export"
	   has no icon everyone reads the same way. */
	.trigger {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		height: 32px;
		padding: 0 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--color-text);
		font-size: var(--text-sm);
		font-weight: 500;
		cursor: pointer;
		transition: background-color var(--speed-fast);
	}

	.trigger:hover,
	.trigger[aria-expanded='true'] {
		background: var(--color-secondary);
	}

	/* The console's menu: a popover surface a step above the page. */
	.menu {
		position: absolute;
		top: calc(100% + 6px);
		right: 0;
		z-index: 30;
		width: 260px;
		padding: var(--space-1);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--dialog-surface);
		box-shadow: var(--shadow-md);
		animation: menu-in var(--speed) ease;
	}

	@keyframes menu-in {
		from {
			opacity: 0;
			transform: translateY(-4px);
		}
	}

	.group {
		display: flex;
		align-items: center;
		gap: 6px;
		margin: 6px 8px 2px;
		color: var(--color-text-hint);
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}

	a {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 34px;
		padding: 0 8px 0 28px;
		border-radius: var(--radius-sm);
		color: var(--color-text);
		font-size: var(--text-sm);
		text-decoration: none;
	}

	a:hover,
	a:focus-visible {
		background: var(--color-secondary);
		outline: none;
	}

	a span {
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: 11px;
	}

	.hint {
		margin: var(--space-1) 0 0;
		padding: var(--space-2) 8px var(--space-1);
		border-top: 1px solid var(--color-border);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
		line-height: 1.45;
	}

	.hint strong {
		color: var(--color-text);
		font-weight: 500;
	}

	@media (max-width: 40rem) {
		.label {
			display: none;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.menu {
			animation: none;
		}
	}
</style>
