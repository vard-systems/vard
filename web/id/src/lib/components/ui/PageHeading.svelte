<script lang="ts">
	type Props = {
		title: string;
		description?: string;
	};

	let { title, description }: Props = $props();
</script>

<div class="heading">
	<h1>{title}</h1>
	{#if description}<p>{description}</p>{/if}
</div>

<style>
	h1 {
		font-size: var(--text-2xl);
		line-height: 1.2;
	}

	p {
		margin-top: var(--space-1);
		color: var(--color-text-hint);
	}
</style>
