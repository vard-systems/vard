<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<!-- Two fields side by side, one above the other on a narrow screen. -->
<div class="row">
	{@render children()}
</div>

<style>
	.row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-4);
	}

	@media (max-width: 30rem) {
		.row {
			grid-template-columns: 1fr;
		}
	}
</style>
