<script lang="ts">
	import { RiArrowRightSLine } from 'svelte-remixicon';
	import type { NavPage } from '#lib/docs/types.js';
	import { iconFor } from '#lib/docs/icons.js';
	import Icon from '../Icon.svelte';
	import SidebarLink from './SidebarLink.svelte';

	type Props = {
		id: string;
		title: string;
		icon: string;
		pages: NavPage[];
		isCurrent: (path: string) => boolean;
	};

	let { id, title, icon, pages, isCurrent }: Props = $props();

	const holdsCurrent = $derived(pages.some((p) => isCurrent(p.href)));

	// Open on the branch holding the page being read; otherwise the reader's.
	let chosen = $state<boolean | null>(null);
	const open = $derived(chosen ?? holdsCurrent);
</script>

<button
	type="button"
	class="row"
	aria-expanded={open}
	aria-controls="branch-{id}"
	onclick={() => (chosen = !open)}
>
	<span class="glyph"><Icon icon={iconFor(icon)} /></span>
	<span class="label">{title}</span>
	<span class="chevron" class:open aria-hidden="true"><Icon icon={RiArrowRightSLine} /></span>
</button>

<!-- Kept in the markup while closed, so opening is a height change rather
     than a mount; inert until open. As the console's branches. -->
<div id="branch-{id}" class="pages" class:open aria-hidden={!open} inert={!open}>
	<div class="pages-inner">
		<ul>
			{#each pages as page (page.href)}
				<li>
					<SidebarLink path={page.href} label={page.title} current={isCurrent(page.href)} nested />
				</li>
			{/each}
		</ul>
	</div>
</div>

<style>
	/* Laid out like a link, set a step smaller and heavier: a branch names a
	   subject, and the column reads as headings with their pages. */
	.row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		height: var(--nav-item-height);
		padding: 0 8px 0 10px;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--nav-text);
		font-size: var(--text-sm);
		font-weight: 600;
		white-space: nowrap;
		cursor: pointer;
		transition: background-color var(--speed-fast);
	}

	.row:hover {
		background: var(--nav-hover);
	}

	.row:focus-visible {
		outline: 2px solid var(--color-accent);
		outline-offset: -2px;
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
		text-align: left;
		text-overflow: ellipsis;
	}

	.chevron {
		display: inline-flex;
		color: var(--color-text-hint);
		transition: transform var(--speed);
	}

	.row:hover .chevron {
		color: var(--color-text);
	}

	.chevron.open {
		transform: rotate(90deg);
	}

	/* The pages under their branch, with the rule down the left that makes
	   the column a tree. */
	.pages {
		display: grid;
		grid-template-rows: 0fr;
		margin: 0 0 0 17px;
		visibility: hidden;
		transition:
			grid-template-rows var(--speed) ease,
			visibility 0s linear var(--speed);
	}

	.pages.open {
		grid-template-rows: 1fr;
		margin-top: 1px;
		visibility: visible;
		transition: grid-template-rows var(--speed) ease;
	}

	.pages-inner {
		min-height: 0;
		overflow: hidden;
		padding-left: var(--nav-branch-inset);
		border-left: 1px solid var(--color-border);
	}

	ul {
		display: flex;
		flex-direction: column;
		gap: 1px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	@media (prefers-reduced-motion: reduce) {
		.pages,
		.pages.open,
		.chevron {
			transition: none;
		}
	}
</style>
