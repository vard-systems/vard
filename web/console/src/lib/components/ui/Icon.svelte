<script lang="ts">
	import type { IconData } from '#lib/icons.js';

	type Props = {
		icon: IconData;
		size?: string;
		/** What the icon says, for a screen reader. Without one the icon is decoration and hidden. */
		label?: string;
	};

	let { icon, size = '1rem', label }: Props = $props();
</script>

<svg
	class="icon"
	viewBox="0 0 24 24"
	width={size}
	height={size}
	fill="currentColor"
	role={label ? 'img' : undefined}
	aria-label={label || undefined}
	aria-hidden={label ? undefined : 'true'}
	xmlns="http://www.w3.org/2000/svg"
>
	<path d={icon} />
</svg>

<style>
	.icon {
		flex: none;
		vertical-align: -0.125em;
	}
</style>
