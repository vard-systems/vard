<script lang="ts">
	import type { Snippet } from 'svelte';
	import { fly } from 'svelte/transition';

	import Button from './Button.svelte';

	type Props = {
		count: number;
		noun?: string;
		onReset: () => void;
		children: Snippet;
	};

	let { count, noun = 'record', onReset, children }: Props = $props();
</script>

{#if count > 0}
	<div class="bar" role="status" transition:fly={{ y: 12, duration: 150 }}>
		<span class="count">
			Selected <strong>{count}</strong>
			{count === 1 ? noun : `${noun}s`}
		</span>

		<Button size="sm" variant="subtle" onclick={onReset}>Reset</Button>

		<span class="actions">
			{@render children()}
		</span>
	</div>
{/if}

<style>
	.bar {
		position: sticky;
		bottom: var(--space-4);
		z-index: 20;
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: fit-content;
		max-width: calc(100% - var(--page-gutter) * 2);
		margin: var(--space-4) auto 0;
		padding: var(--space-2);
		padding-left: var(--space-4);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-surface);
		box-shadow: var(--shadow-md);
	}

	.count {
		color: var(--color-text-hint);
		white-space: nowrap;
	}

	.count strong {
		color: var(--color-text);
	}

	.actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-left: var(--space-3);
	}

	.bar :global(.control) {
		border-radius: var(--radius-pill);
	}
</style>
