<script lang="ts">
	import { untrack, type Snippet } from 'svelte';
	import AppHeader from '$lib/components/layout/AppHeader.svelte';
	import { provideShell } from '$lib/state/shell.svelte';
	import type { LayoutData } from './$types';

	let { data, children }: { data: LayoutData; children: Snippet } = $props();

	const shell = provideShell(
		untrack(() => data.sidebar),
		untrack(() => data.closedGroups)
	);
</script>

<div class="shell" class:mini={shell.collapsed}>
	<AppHeader admin={data.admin} organization={data.organization} />

	<main>
		{@render children()}
	</main>
</div>

<style>
	@property --sidebar-width {
		syntax: '<length>';
		inherits: true;
		initial-value: 240px;
	}

	.shell {
		--sidebar-width: 240px;

		height: 100dvh;
		overflow: hidden;
		transition: --sidebar-width var(--speed-slow) cubic-bezier(0.4, 0, 0.2, 1);
	}

	.shell.mini {
		--sidebar-width: 56px;
	}

	@media (prefers-reduced-motion: reduce) {
		.shell {
			transition: none;
		}
	}

	main {
		position: relative;
		height: 100%;
		overflow-y: auto;
		overscroll-behavior-y: auto;
		padding: calc(var(--header-height) + var(--space-4)) 0 var(--space-4);
		scrollbar-gutter: stable;
	}

	main:has(:global(.dashboard)) {
		padding: var(--header-height) 0 0;
	}
</style>
