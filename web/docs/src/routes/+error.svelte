<script lang="ts">
	import { page } from '$app/state';
	import { href } from '#lib/docs/links.js';
</script>

<svelte:head>
	<title>Not found</title>
</svelte:head>

<div class="error">
	<p class="status">{page.status}</p>
	<h1>{page.status === 404 ? 'There is no page here.' : 'Something went wrong.'}</h1>
	<p>Try the search above, or start from the beginning.</p>
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- href() is resolve() -->
	<a href={href('/')}>Go to the introduction</a>
</div>

<style>
	.error {
		max-width: 32rem;
		margin: 6rem auto;
		padding: 0 1rem;
		text-align: center;
	}

	.status {
		margin: 0;
		font-family: var(--font-mono);
		color: var(--accent-text);
		font-weight: 700;
	}

	h1 {
		margin: 0.5rem 0;
	}

	p {
		color: var(--text-muted);
	}
</style>
