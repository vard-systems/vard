<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from './icon';

	type Props = {
		tone?: 'neutral' | 'success' | 'warning' | 'danger';
		icon?: IconName;
		children: Snippet;
	};

	let { tone = 'neutral', icon, children }: Props = $props();
</script>

<span class="badge {tone}">
	{#if icon}<i class={icon} aria-hidden="true"></i>{/if}
	{@render children()}
</span>

<style>
	.badge {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		justify-self: start;
		height: 25px;
		padding: 0 7px;
		border-radius: var(--radius-control);
		font-size: var(--text-sm);
		white-space: nowrap;
	}

	.neutral {
		background: var(--color-secondary);
		color: var(--color-text-hint);
	}

	.success {
		background: var(--surface-success);
		color: var(--color-success);
	}

	.warning {
		background: var(--surface-warning);
		color: var(--color-warning);
	}

	.danger {
		background: var(--surface-danger);
		color: var(--color-danger);
	}
</style>
