<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		children: Snippet;
	};

	let { children }: Props = $props();
</script>

<p class="note">{@render children()}</p>

<style>
	.note {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.5;
	}

	.note :global(a),
	.note :global(strong) {
		color: var(--color-text);
	}

	/* An action that belongs to the sentence, such as undoing what it says. */
	.note :global(button) {
		margin-left: var(--space-1);
		padding: 0;
		border: none;
		background: none;
		color: var(--color-text);
		font: inherit;
		text-decoration: underline;
		text-underline-offset: 3px;
		cursor: pointer;
	}
</style>
