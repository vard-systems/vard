<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Button } from '#lib/components/ui/index.js';

	type Props = {
		/** How many are picked: the add button counts them, and waits for one. */
		count: number;
		oncancel: () => void;
		onadd: () => void;
		children: Snippet;
	};

	let { count, oncancel, onadd, children }: Props = $props();
</script>

<div class="panel">
	{@render children()}

	<div class="foot">
		<Button size="sm" variant="subtle" onclick={oncancel}>Cancel</Button>
		<Button size="sm" onclick={onadd} disabled={count === 0}>
			Add{count > 0 ? ` ${count}` : ''}
		</Button>
	</div>
</div>

<style>
	.panel {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		background: var(--color-surface);
	}

	.foot {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
