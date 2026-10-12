<script lang="ts">
	import type { ComponentType } from 'svelte';

	type Props = {
		icon: ComponentType;
		size?: string;
		label?: string;
	};

	let { icon: Glyph, size = '1rem', label }: Props = $props();
</script>

<Glyph {size} {label} class="icon" />

<style>
	:global(.icon) {
		flex: none;
		vertical-align: -0.125em;
	}
</style>
