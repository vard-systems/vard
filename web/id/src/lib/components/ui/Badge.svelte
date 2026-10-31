<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from './icon';

	type Props = {
		tone?: 'neutral' | 'success';
		size?: 'sm' | 'md';
		icon?: IconName;
		children: Snippet;
	};

	let { tone = 'neutral', size = 'sm', icon, children }: Props = $props();
</script>

<span class={['badge', tone, size]}>
	{#if icon}<i class={icon} style:font-size="13px" aria-hidden="true"></i>{/if}
	{@render children()}
</span>

<style>
	.badge {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		border-radius: 999px;
		background: var(--color-input);
		color: var(--color-text-hint);
		font-weight: 600;
		white-space: nowrap;
	}

	.sm {
		padding: 1px 8px;
		font-size: var(--text-xs);
	}

	.md {
		padding: 2px 10px;
		font-size: var(--text-base);
	}

	.success {
		background: color-mix(in srgb, var(--color-success), transparent 86%);
		color: var(--color-success);
	}
</style>
