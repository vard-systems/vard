<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Note } from 'ui';

	type Props = {
		align?: 'start' | 'center';
		children: Snippet;
	};

	let { align = 'start', children }: Props = $props();
</script>

<div class={['note', align]}><Note>{@render children()}</Note></div>

<style>
	.center {
		text-align: center;
	}
</style>
