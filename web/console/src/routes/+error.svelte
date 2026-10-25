<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { pageTitle } from '#lib/brand.js';
</script>

<svelte:head>
	<title>{pageTitle(String(page.status))}</title>
</svelte:head>

<main>
	<h1>{page.status}</h1>
	<p>{page.error?.message ?? 'Something went wrong.'}</p>
	<a href={resolve('/(panel)')}>Back to the admin panel</a>
</main>

<style>
	main {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		place-content: center;
		align-items: center;
		min-height: 100dvh;
		padding: var(--space-5);
		text-align: center;
	}

	p {
		color: var(--color-text-hint);
	}
</style>
