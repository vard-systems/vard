<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Panel } from 'ui';

	type Props = {
		title: string;
		description?: string;
		children: Snippet;
		footer?: Snippet;
		aside?: Snippet;
	};

	let { title, description, children, footer, aside }: Props = $props();
</script>

<Panel {title} meta={aside}>
	<div class="panel">
		{#if description}<p class="description">{description}</p>{/if}
		<div class="body">{@render children()}</div>
		{#if footer}<footer>{@render footer()}</footer>{/if}
	</div>
</Panel>

<style>
	.panel {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.description {
		color: var(--color-text-hint);
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	footer {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
