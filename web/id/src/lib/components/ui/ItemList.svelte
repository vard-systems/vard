<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = { label?: string; children: Snippet };

	let { label, children }: Props = $props();
</script>

<ul aria-label={label}>{@render children()}</ul>

<style>
	ul {
		display: flex;
		flex-direction: column;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	ul > :global(li) {
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--color-border);
	}

	ul > :global(li:first-child) {
		padding-top: 0;
	}

	ul > :global(li:last-child) {
		padding-bottom: 0;
		border-bottom: 0;
	}
</style>
