<script lang="ts">
	import { beforeNavigate } from '$app/navigation';
	import { updated } from '$app/state';
	import { Toaster } from '#lib/components/index.js';
	import { provideTranslator, translator } from '#lib/i18n/index.js';
	import '#lib/styles/app.css';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	provideTranslator(
		translator(
			() => data.messages,
			() => data.language
		)
	);

	$effect(() => {
		document.documentElement.lang = data.language;
	});

	// After a deploy, a tab that was already open still runs the old code;
	// the next link it follows loads the new version whole instead.
	beforeNavigate(({ willUnload, to }) => {
		if (updated.current && !willUnload && to?.url) location.href = to.url.href;
	});
</script>

{@render children()}

<Toaster />
