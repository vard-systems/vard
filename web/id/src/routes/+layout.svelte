<script lang="ts">
	import { Toaster } from '$lib/components';
	import { provideTranslator, translator } from '$lib/i18n';
	import '$lib/styles/app.css';
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
