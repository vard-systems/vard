<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type Props = HTMLButtonAttributes & {
		variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
		size?: 'md' | 'sm';
		loading?: boolean;
		block?: boolean;
		children: Snippet;
	};

	let {
		variant = 'primary',
		size = 'md',
		loading = false,
		block = false,
		disabled = false,
		type = 'button',
		children,
		...rest
	}: Props = $props();
</script>

<button
	{type}
	class="button {variant} {size}"
	class:block
	disabled={disabled || loading}
	aria-busy={loading || undefined}
	{...rest}
>
	{#if loading}<span class="spinner" aria-hidden="true"></span>{/if}
	{@render children()}
</button>

<style>
	.button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		height: var(--control-height);
		padding: 0 var(--space-5);
		border: 1px solid transparent;
		border-radius: var(--radius-md);
		font-weight: 600;
		font-size: var(--text-base);
		white-space: nowrap;
		cursor: pointer;
		transition:
			background var(--speed),
			border-color var(--speed),
			opacity var(--speed);
	}

	.sm {
		height: 36px;
		padding: 0 var(--space-4);
		font-size: var(--text-sm);
	}

	.block {
		width: 100%;
	}

	.primary {
		background: var(--color-primary);
		color: var(--color-primary-text);
	}

	.primary:hover:not(:disabled) {
		background: var(--color-primary-hover);
	}

	.secondary {
		background: var(--color-surface);
		border-color: var(--color-border-strong);
		color: var(--color-text);
	}

	.secondary:hover:not(:disabled) {
		background: var(--color-surface-alt);
	}

	.danger {
		background: transparent;
		border-color: color-mix(in srgb, var(--color-danger), transparent 55%);
		color: var(--color-danger);
	}

	.danger:hover:not(:disabled) {
		background: color-mix(in srgb, var(--color-danger), transparent 90%);
	}

	.ghost {
		background: transparent;
		color: var(--color-text-hint);
	}

	.ghost:hover:not(:disabled) {
		background: var(--color-input);
		color: var(--color-text);
	}

	.button:disabled {
		cursor: not-allowed;
		opacity: 0.55;
	}

	.button[aria-busy='true'] {
		opacity: 0.85;
	}

	.spinner {
		width: 1em;
		height: 1em;
		border: 2px solid currentColor;
		border-right-color: transparent;
		border-radius: 50%;
		animation: spin 0.7s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
