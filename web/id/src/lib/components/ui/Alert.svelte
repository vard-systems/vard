<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		tone?: 'danger' | 'success' | 'info' | 'warning';
		children: Snippet;
	};

	let { tone = 'danger', children }: Props = $props();

	const toneIcon = $derived(
		tone === 'success'
			? 'ri-checkbox-circle-line'
			: tone === 'info'
				? 'ri-information-line'
				: 'ri-error-warning-line'
	);
</script>

<div class="alert {tone}" role={tone === 'danger' ? 'alert' : 'status'}>
	<i class={toneIcon} aria-hidden="true"></i>
	<div class="body">{@render children()}</div>
</div>

<style>
	.alert {
		--tone: var(--color-danger);
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		padding: var(--space-3);
		border: 1px solid color-mix(in srgb, var(--tone), transparent 70%);
		border-radius: var(--radius-md);
		background: color-mix(in srgb, var(--tone), transparent 91%);
		font-size: var(--text-base);
		line-height: 1.5;
	}

	.alert :global(i) {
		margin-top: 1px;
		color: var(--tone);
	}

	.body {
		min-width: 0;
		overflow-wrap: anywhere;
	}

	.danger {
		color: var(--color-danger);
	}

	.success {
		--tone: var(--color-success);
	}

	.info {
		--tone: var(--color-info);
	}

	.warning {
		--tone: var(--color-warning);
	}
</style>
