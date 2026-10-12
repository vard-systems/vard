<script lang="ts">
	import type { ComponentType } from 'svelte';
	import Icon from './Icon.svelte';

	type Props = { label: string; icon?: ComponentType; required?: boolean };

	let { label, icon, required = false }: Props = $props();
</script>

{#if icon}<Icon {icon} size="1em" />{/if}{label}{#if required}<span
		data-part="required-indicator"
		aria-hidden="true">&thinsp;*</span
	>{/if}
