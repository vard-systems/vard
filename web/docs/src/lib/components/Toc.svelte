<script lang="ts">
	import type { TocEntry } from '#lib/docs/types.js';

	let { entries }: { entries: TocEntry[] } = $props();

	let active = $state('');

	// The heading nearest the top of the screen is the one being read.
	$effect(() => {
		const headings = entries
			.map((entry) => document.getElementById(entry.id))
			.filter((element): element is HTMLElement => element !== null);
		if (headings.length === 0) return;

		const observer = new IntersectionObserver(
			(seen) => {
				const visible = seen.filter((e) => e.isIntersecting).map((e) => e.target.id);
				if (visible.length > 0) active = visible[0];
			},
			{ rootMargin: '-60px 0px -70% 0px' }
		);
		headings.forEach((heading) => observer.observe(heading));
		return () => observer.disconnect();
	});
</script>

{#if entries.length > 1}
	<nav aria-label="On this page">
		<h2 class="group-label">On this page</h2>
		<ul>
			{#each entries as entry (entry.id)}
				<li class:nested={entry.depth === 3}>
					<a href="#{entry.id}" class:active={active === entry.id}>
						{#if entry.method}
							<span class="method method-{entry.method.toLowerCase()}">{entry.method}</span>
						{/if}
						<span class:path={entry.method}>{entry.text}</span>
					</a>
				</li>
			{/each}
		</ul>
	</nav>
{/if}

<style>
	nav {
		position: sticky;
		top: calc(var(--header-height) + var(--space-4));
		max-height: calc(100dvh - var(--header-height) - var(--space-6));
		overflow-y: auto;
		scrollbar-width: none;
	}

	.group-label {
		margin: 0 0 var(--space-1);
		color: var(--color-text-hint);
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}

	/* The tree's rule, as under a sidebar branch; the heading being read
	   marks it in the brand colour. */
	ul {
		margin: 0;
		padding: 0;
		list-style: none;
		border-left: 1px solid var(--color-border);
	}

	li.nested {
		padding-left: var(--space-2);
	}

	a {
		display: flex;
		align-items: center;
		gap: 6px;
		margin-left: -1px;
		padding: 4px 0 4px var(--space-2);
		border-left: 2px solid transparent;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.35;
		text-decoration: none;
	}

	a:hover {
		color: var(--color-text);
	}

	a.active {
		border-left-color: var(--color-brand);
		color: var(--color-text);
	}

	.method {
		min-width: 40px;
		height: 16px;
		font-size: 9.5px;
	}

	.path {
		overflow: hidden;
		font-family: var(--font-mono);
		font-size: var(--text-xs);
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
