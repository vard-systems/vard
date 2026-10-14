<script lang="ts">
	import { href } from '#lib/docs/links.js';
	import { iconFor } from '#lib/docs/icons.js';
	import Icon from '../Icon.svelte';

	type Props = {
		path: string;
		label: string;
		/** A Remix Icon name; a page under a branch has none. */
		icon?: string;
		current: boolean;
		/** Under a branch: against the tree's rule, which it marks when it is
		    the page being read. */
		nested?: boolean;
	};

	let { path, label, icon, current, nested = false }: Props = $props();
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -- href() is resolve() -->
<a
	href={href(path)}
	class="row"
	class:current
	class:nested
	aria-current={current ? 'page' : undefined}
>
	{#if icon && !nested}
		<span class="glyph"><Icon icon={iconFor(icon)} /></span>
	{/if}
	<span class="label">{label}</span>
</a>

<!-- eslint-enable svelte/no-navigation-without-resolve -->

<style>
	/* The console's row: a name beside its icon, filled in the brand colour
	   when it is the page being read — the one blue thing in the column. */
	.row {
		position: relative;
		display: flex;
		align-items: center;
		gap: var(--space-2);
		height: var(--nav-item-height);
		padding: 0 10px;
		border-radius: var(--radius-sm);
		color: var(--nav-text);
		font-size: var(--text-base);
		white-space: nowrap;
		text-decoration: none;
		transition:
			background-color var(--speed-fast),
			color var(--speed-fast);
	}

	.row:hover {
		background: var(--nav-hover);
	}

	.row:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: -2px;
	}

	.row.current {
		background: var(--nav-current);
		color: var(--nav-current-text);
		font-weight: 500;
	}

	.glyph {
		display: inline-flex;
		flex: none;
		width: 16px;
		height: 16px;
		align-items: center;
		justify-content: center;
	}

	.label {
		flex: 1 1 0;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* Under a branch the row is a step shorter, and the page being read also
	   marks the tree's rule beside it. */
	.row.nested {
		height: calc(var(--nav-item-height) - 4px);
		padding-left: 8px;
		font-size: var(--text-sm);
	}

	.row.nested.current::before {
		content: '';
		position: absolute;
		top: 50%;
		left: calc(var(--nav-branch-inset) * -1);
		width: 3px;
		height: calc(100% - 8px);
		border-radius: var(--radius-pill);
		background: var(--nav-mark);
		transform: translateY(-50%);
	}
</style>
