<script lang="ts">
	/** The page's <title>: its parts, most specific first, the empty ones left out. */
	let { parts }: { parts: (string | null | undefined)[] } = $props();

	const title = $derived(parts.filter(Boolean).join(' · '));
</script>

<svelte:head>
	<title>{title}</title>
</svelte:head>
