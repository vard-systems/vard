<script lang="ts">
	import type { Snippet } from 'svelte';
	import { RiErrorWarningLine, RiInformationLine, RiCheckboxCircleLine } from '#lib/icons.js';
	import type { ColorPalette } from './control';
	import Icon from './Icon.svelte';

	type Props = {
		tone?: Exclude<ColorPalette, 'neutral'>;
		children: Snippet;
	};

	let { tone = 'danger', children }: Props = $props();

	const icons = {
		danger: RiErrorWarningLine,
		warning: RiErrorWarningLine,
		info: RiInformationLine,
		success: RiCheckboxCircleLine
	};
</script>

<!-- An error is announced as soon as it appears; a notice is just read in its
     place. -->
<p class="alert {tone}" role={tone === 'danger' ? 'alert' : 'status'}>
	<Icon icon={icons[tone]} size="1.0625rem" />
	<span>{@render children()}</span>
</p>

<style>
	.alert {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		margin: 0;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius-surface);
		font-size: var(--text-base);
		line-height: 1.5;
	}

	.alert :global(svg) {
		flex-shrink: 0;
		margin-top: 2px;
	}

	.danger {
		background: var(--surface-danger);
		color: var(--color-danger);
	}

	.warning {
		background: var(--surface-warning);
		color: var(--color-text);
	}

	.warning :global(svg) {
		color: var(--color-warning);
	}

	.info {
		background: var(--surface-info);
		color: var(--color-text);
	}

	.info :global(svg) {
		color: var(--color-info);
	}

	.success {
		background: var(--surface-success);
		color: var(--color-text);
	}

	.success :global(svg) {
		color: var(--color-success);
	}
</style>
