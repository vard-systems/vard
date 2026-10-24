<script lang="ts">
	import { resolve } from '$app/paths';
	import { TEMPLATES } from './steps';
</script>

<section class="templates" aria-label="Start from a template">
	<h2>Start from a template</h2>
	<div class="cards">
		{#each TEMPLATES as template (template.id)}
			<!-- The path is resolved; only the query is added to it. -->
			<!-- eslint-disable svelte/no-navigation-without-resolve -->
			<a class="card" href={`${resolve('/(panel)/flows/new')}?template=${template.id}`}>
				<span class="mark"
					><i class={template.icon} style:font-size="19px" aria-hidden="true"></i></span
				>
				<strong>{template.name}</strong>
				<span>{template.hint}</span>
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/each}
	</div>
</section>

<style>
	.templates h2 {
		margin: 0 0 var(--space-2);
		font-size: var(--text-base);
	}

	.cards {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: var(--space-3);
	}

	.card {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		padding: var(--space-4);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		background: var(--color-surface);
		color: var(--color-text);
		text-decoration: none;
		transition:
			border-color var(--speed-fast),
			background-color var(--speed-fast);
	}

	.card:hover {
		border-color: var(--color-info);
		background: var(--surface-info);
	}

	.card span:last-child {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.mark {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		margin-bottom: var(--space-1);
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
	}
</style>
