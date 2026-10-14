<script lang="ts">
	import '#lib/styles/app.css';
	import { afterNavigate } from '$app/navigation';
	import favicon from '#lib/assets/favicon.svg';
	import { THEME_KEY } from '#lib/brand.js';
	import AppHeader from '#lib/components/layout/AppHeader.svelte';
	import Sidebar from '#lib/components/layout/Sidebar.svelte';

	let { data, children } = $props();

	let menuOpen = $state(false);

	afterNavigate(({ shallow, type }) => {
		if (shallow && type === 'goto') return;

		return (menuOpen = false);
	});

	const themeScript =
		`<script>try{var t=localStorage.getItem(${JSON.stringify(THEME_KEY)});` +
		`if(t)document.documentElement.dataset.theme=t}catch(e){}</` +
		`script>`;
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<!-- eslint-disable-next-line svelte/no-at-html-tags -- a constant written above, not content -->
	{@html themeScript}
</svelte:head>

<svelte:window onkeydown={(event) => event.key === 'Escape' && (menuOpen = false)} />

<a class="skip" href="#content">Skip to content</a>

<AppHeader {menuOpen} onMenu={() => (menuOpen = !menuOpen)} />

<Sidebar sections={data.navigation} open={menuOpen} onClose={() => (menuOpen = false)} />

<main id="content">{@render children()}</main>

<style>
	.skip {
		position: absolute;
		top: var(--space-2);
		left: -999px;
		z-index: 100;
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		background: var(--color-brand);
		color: var(--color-brand-text);
	}

	.skip:focus {
		left: var(--space-2);
	}

	main {
		min-width: 0;
		padding-top: var(--header-height);
		padding-left: var(--sidebar-width);
	}

	@media (max-width: 55rem) {
		main {
			padding-left: 0;
		}
	}
</style>
