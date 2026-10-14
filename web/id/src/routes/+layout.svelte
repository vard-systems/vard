<script lang="ts">
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
</script>

{@render children()}

<Toaster />
