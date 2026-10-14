<script lang="ts">
	import type { Snippet } from 'svelte';
	import Sidebar from '#lib/components/layout/Sidebar.svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="dashboard">
	<Sidebar />

	<div class="content">
		{@render children()}
	</div>
</div>

<style>
	.dashboard {
		display: grid;
		grid-template-columns: var(--sidebar-width) 1fr;
		align-items: start;
	}

	.content {
		grid-column: 2;
		box-sizing: border-box;
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		width: 100%;
		min-width: 0;
		max-width: calc(var(--content-max-width) + var(--page-gutter) * 2);
		margin-inline: auto;
		padding: var(--space-5) var(--page-gutter) var(--space-6);
	}

	@media (max-width: 55rem) {
		.dashboard {
			grid-template-columns: 1fr;
		}

		.content {
			grid-column: auto;
		}
	}
</style>
