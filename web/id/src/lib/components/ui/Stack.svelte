<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		gap?: 'sm' | 'md' | 'lg';
		children: Snippet;
	};

	let { gap = 'md', children }: Props = $props();
</script>

<div class={['stack', gap]}>
	{@render children()}
</div>

<style>
	.stack {
		display: flex;
		flex-direction: column;
	}

	.sm {
		gap: var(--space-3);
	}

	.md {
		gap: var(--space-4);
	}

	.lg {
		gap: var(--space-5);
	}
</style>
