<script lang="ts">
	import { page } from '$app/state';
	import { href } from '#lib/docs/links.js';
	import type { NavSection } from '#lib/docs/types.js';
	import SidebarBranch from './SidebarBranch.svelte';
	import SidebarLink from './SidebarLink.svelte';

	type Props = {
		sections: NavSection[];
		/** Open as a panel over the page, on a screen too narrow for a column. */
		open: boolean;
		onClose: () => void;
	};

	let { sections, open, onClose }: Props = $props();

	const current = $derived(page.url.pathname.replace(/\/$/, '') || '/');
	const isCurrent = (path: string) => (href(path).replace(/\/$/, '') || '/') === current;
</script>

{#if open}
	<!-- svelte-ignore a11y_click_events_have_key_events -->
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div class="backdrop" onclick={onClose}></div>
{/if}

<aside class:open>
	<nav aria-label="Documentation">
		{#each sections as section, i (section.label)}
			<section class:ruled={i < sections.length - 1}>
				<h2 class="group-label">{section.label}</h2>
				<ul>
					{#each section.items as item (item.kind === 'link' ? item.href : item.id)}
						<li>
							{#if item.kind === 'link'}
								<SidebarLink
									path={item.href}
									label={item.title}
									icon={item.icon}
									current={isCurrent(item.href)}
								/>
							{:else}
								<SidebarBranch
									id={item.id.replace(/\//g, '-')}
									title={item.title}
									icon={item.icon}
									pages={item.pages}
									{isCurrent}
								/>
							{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/each}
	</nav>
</aside>

<style>
	/* The console's column, and the names every row is drawn with. */
	aside {
		--nav-text: var(--color-text);
		--nav-hover: var(--color-secondary);
		--nav-current: var(--color-brand);
		--nav-current-text: var(--color-brand-text);
		--nav-mark: var(--color-brand);
		--nav-branch-inset: 8px;

		position: fixed;
		top: var(--header-height);
		bottom: 0;
		left: 0;
		z-index: 9;
		display: flex;
		flex-direction: column;
		width: var(--sidebar-width);
		border-right: 1px solid var(--color-border);
		background: var(--color-surface);
	}

	nav {
		flex: 1;
		padding: var(--space-2) var(--space-2) var(--space-5);
		overflow-y: auto;
		overscroll-behavior: contain;
		scrollbar-width: none;
	}

	nav::-webkit-scrollbar {
		display: none;
	}

	/* A labelled group, ruled off from the next, like Overview and Manage. */
	section.ruled {
		padding-bottom: var(--space-2);
		margin-bottom: var(--space-1);
		border-bottom: 1px solid var(--color-border);
	}

	.group-label {
		margin: 6px 10px 4px;
		color: var(--color-text-hint);
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.06em;
		line-height: 16px;
		text-transform: uppercase;
	}

	ul {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	/* On a screen too narrow for a column, the same rows slide in from the
	   edge over a dimmed page. A media query, so the page is served with the
	   panel already off the edge. */
	@media (max-width: 55rem) {
		aside {
			width: min(17rem, 82vw);
			box-shadow: var(--shadow-md);
			transform: translateX(-100%);
			visibility: hidden;
			transition:
				transform var(--speed-slow) cubic-bezier(0.4, 0, 0.2, 1),
				visibility var(--speed-slow);
		}

		aside.open {
			transform: none;
			visibility: visible;
		}
	}

	.backdrop {
		position: fixed;
		inset: var(--header-height) 0 0 0;
		z-index: 8;
		background: var(--color-overlay);
	}

	@media (prefers-reduced-motion: reduce) {
		aside {
			transition: none;
		}
	}
</style>
