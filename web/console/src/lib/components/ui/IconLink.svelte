<script lang="ts">
	import type { ComponentType } from 'svelte';
	import type { HTMLAnchorAttributes } from 'svelte/elements';
	import type { ControlProps, Size } from './control';
	import Icon from './Icon.svelte';
	import Tooltip from './Tooltip.svelte';

	type Props = HTMLAnchorAttributes &
		Omit<ControlProps, 'icon' | 'loading' | 'disabled'> & {
			href: string;
			icon: ComponentType;
			label: string;
			placement?: 'top' | 'bottom' | 'left' | 'right';
		};

	let {
		href,
		icon,
		label,
		size = 'md',
		variant = 'ghost',
		colorPalette = 'neutral',
		placement = 'bottom',
		...rest
	}: Props = $props();

	const glyph: Record<Size, string> = { sm: '1rem', md: '1.125rem', lg: '1.25rem' };
</script>

<!-- eslint-disable svelte/no-navigation-without-resolve -->
<Tooltip {label} {placement}>
	{#snippet children(trigger)}
		<a
			{...trigger()}
			{href}
			class="control"
			data-icon="true"
			data-size={size}
			data-variant={variant}
			data-palette={colorPalette}
			aria-label={label}
			{...rest}
		>
			<Icon {icon} size={glyph[size]} />
		</a>
	{/snippet}
</Tooltip>

<!-- eslint-enable svelte/no-navigation-without-resolve -->
