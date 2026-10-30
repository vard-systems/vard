<script lang="ts">
	import { CopyButton } from 'ui';
	import { highlighted } from './highlight.svelte';

	type Props = {
		code: string;
		title?: string;
		copy?: string;
	};

	let { code, title, copy = title ?? 'code' }: Props = $props();

	const highlight = highlighted(() => code);
</script>

<div class="code-block" class:titled={title}>
	{#if title}
		<div class="head">
			<span>{title}</span>
			<CopyButton value={code} label={copy} />
		</div>
	{:else}
		<span class="corner"><CopyButton value={code} label={copy} /></span>
	{/if}

	<div class="body">
		{#if highlight.html}
			<!-- Shiki's own output, from code it escaped itself. -->
			<!-- eslint-disable-next-line svelte/no-at-html-tags -->
			{@html highlight.html}
		{:else}
			<pre class="shiki"><code>{code}</code></pre>
		{/if}
	</div>
</div>

<style>
	.code-block {
		position: relative;
		min-width: 0;
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		background: var(--color-surface-alt);
	}

	.head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
		padding: 4px 4px 4px var(--space-3);
		border-bottom: 1px solid var(--color-border);
		background: var(--color-secondary);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.corner {
		position: absolute;
		top: 6px;
		right: 6px;
		z-index: 1;
	}

	.body {
		overflow: auto;
	}

	.body :global(pre.shiki) {
		padding: var(--space-3) var(--space-4);
		line-height: 1.6;
	}

	.code-block:not(.titled) .body :global(pre.shiki) {
		padding-right: calc(var(--control-height) + var(--space-2));
	}
</style>
