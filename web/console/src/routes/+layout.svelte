<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { updated } from '$app/state';
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import NavigationProgress from '#lib/components/layout/NavigationProgress.svelte';
	import { Toaster } from '#lib/components/ui/index.js';
	import { createQueryClient } from '#lib/query/index.js';
	import '#lib/styles/app.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// One cache for this visitor, for as long as the tab is open.
	const queryClient = createQueryClient();

	// After a deploy, a tab that was already open still runs the old code;
	// the next link it follows loads the new version whole instead.
	beforeNavigate(({ willUnload, to }) => {
		if (updated.current && !willUnload && to?.url) location.href = to.url.href;
	});
</script>

<NavigationProgress />

<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>

<Toaster />
