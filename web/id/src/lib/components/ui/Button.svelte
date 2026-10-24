<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes, HTMLButtonAttributes } from 'svelte/elements';

	type Props = Omit<HTMLButtonAttributes, 'type'> & {
		variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
		size?: 'md' | 'sm';
		type?: 'button' | 'submit';
		loading?: boolean;
		block?: boolean;
		/** Makes it a link that looks like a button: somewhere to go, not
		    something to do. Resolved by the caller. */
		href?: string;
		target?: HTMLAnchorAttributes['target'];
		rel?: HTMLAnchorAttributes['rel'];
		children: Snippet;
	};

	let {
		variant = 'primary',
		size = 'md',
		type = 'button',
		loading = false,
		block = false,
		disabled = false,
		href,
		target,
		rel,
		children,
		...rest
	}: Props = $props();

	const classes = $derived(['button', variant, size, { block }]);
</script>

{#if href !== undefined}
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the caller resolves it -->
	<a class={classes} {href} {target} {rel}>{@render children()}</a>
{:else}
	<button
		{type}
		class={classes}
		disabled={disabled || loading}
		aria-busy={loading || undefined}
		{...rest}
	>
		{#if loading}<span class="spinner" aria-hidden="true"></span>{/if}
		{@render children()}
	</button>
{/if}

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
		text-decoration: none;
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

	a.button:hover {
		text-decoration: none;
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
