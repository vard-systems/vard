<script lang="ts">
	import type { IconData } from '#lib/icons.js';
	import Icon from './Icon.svelte';

	type Props = {
		src?: string;
		icon?: IconData;
		text?: string;
		size?: 'xs' | 'sm' | 'md';
		shape?: 'square' | 'circle';
		tone?: 'neutral' | 'accent';
		bare?: boolean;
	};

	let {
		src,
		icon,
		text,
		size = 'sm',
		shape = 'square',
		tone = 'neutral',
		bare = false
	}: Props = $props();

	let failed = $state('');

	const picture = $derived(src && src !== failed ? src : '');

	const glyph = { xs: '0.875rem', sm: '1rem', md: '1.15rem' };
</script>

<span
	class="thumb {size} {shape} {tone}"
	class:picture
	class:bare={bare && picture}
	aria-hidden="true"
>
	{#if picture}
		<img src={picture} alt="" onerror={() => (failed = picture)} />
	{:else if icon}
		<Icon {icon} size={glyph[size]} />
	{:else if text}
		{text}
	{/if}
</span>

<style>
	.thumb {
		display: inline-flex;
		flex-shrink: 0;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		aspect-ratio: 1;
		border: 1px solid var(--color-secondary-alt);
		border-radius: var(--radius-md);
		background: var(--color-surface-alt);
		color: var(--color-text-hint);
		font-size: 11px;
		font-weight: 600;
		letter-spacing: 0.02em;
	}

	.xs {
		width: 28px;
		font-size: 10px;
	}

	.sm {
		width: 34px;
	}

	.md {
		width: var(--control-height);
		font-size: var(--text-sm);
	}

	.circle {
		border-radius: var(--radius-pill);
	}

	.accent {
		border-color: transparent;
		background: var(--color-accent);
		color: var(--color-accent-text);
		font-weight: 700;
	}

	.picture {
		border-color: var(--color-border);
		background: var(--color-surface-alt);
	}

	.picture.bare {
		border: none;
		border-radius: 0;
		background: transparent;
	}

	.picture.bare.circle {
		border-radius: var(--radius-pill);
	}

	img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}
</style>
