<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		title: string;
		description?: string;
		children: Snippet;
		footer?: Snippet;
		aside?: Snippet;
	};

	let { title, description, children, footer, aside }: Props = $props();
	const uid = $props.id();
</script>

<section class="panel" aria-labelledby="panel-{uid}">
	<header>
		<div>
			<h2 id="panel-{uid}">{title}</h2>
			{#if description}<p>{description}</p>{/if}
		</div>
		{#if aside}{@render aside()}{/if}
	</header>

	<div class="body">
		{@render children()}
	</div>

	{#if footer}
		<footer>{@render footer()}</footer>
	{/if}
</section>

<style>
	.panel {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		background: var(--color-surface);
		box-shadow: var(--shadow-card);
	}

	header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		padding: var(--space-5) var(--space-5) 0;
	}

	h2 {
		font-size: var(--text-lg);
		font-weight: 700;
	}

	header p {
		margin-top: var(--space-1);
		color: var(--color-text-hint);
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		padding: var(--space-5);
	}

	footer {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-5);
		border-top: 1px solid var(--color-border);
		background: var(--color-surface-alt);
		border-radius: 0 0 var(--radius-lg) var(--radius-lg);
	}

	@media (max-width: 30rem) {
		header,
		.body {
			padding-left: var(--space-4);
			padding-right: var(--space-4);
		}

		footer {
			padding: var(--space-3) var(--space-4);
		}

		footer :global(.button) {
			flex: 1;
		}
	}
</style>
