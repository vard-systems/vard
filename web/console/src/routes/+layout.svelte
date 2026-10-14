<script lang="ts">
	import { QueryClientProvider } from '@tanstack/svelte-query';
	import NavigationProgress from '#lib/components/layout/NavigationProgress.svelte';
	import { Toaster } from '#lib/components/ui/index.js';
	import { createQueryClient } from '#lib/query/index.js';
	import '#lib/styles/app.css';
	import type { LayoutProps } from './$types';

	let { children }: LayoutProps = $props();

	// One cache for this visitor, built here so that rendering on the server
	// gives each request its own.
	const queryClient = createQueryClient();
</script>

<!-- No <title> here on purpose. A title in this layout is only applied once
     the app has booted, so it lands between the one the browser already has
     and the one the page sets, which shows as a second flicker. The default
     lives in app.html instead, where it is in the very first byte. -->
<NavigationProgress />

<QueryClientProvider client={queryClient}>
	{@render children()}
</QueryClientProvider>

<Toaster />
