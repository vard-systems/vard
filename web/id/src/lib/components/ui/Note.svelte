<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		align?: 'start' | 'center';
		size?: 'sm' | 'md';
		children: Snippet;
	};

	let { align = 'start', size = 'md', children }: Props = $props();
</script>

<p class={['note', align, size]}>{@render children()}</p>

<style>
	.note {
		color: var(--color-text-hint);
	}

	.center {
		text-align: center;
	}

	.sm {
		font-size: var(--text-sm);
	}
</style>
