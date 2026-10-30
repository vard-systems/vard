<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLAnchorAttributes, HTMLButtonAttributes } from 'svelte/elements';
	import { Button, LinkButton, type ColorPalette, type Variant } from 'ui';

	type Props = Omit<HTMLButtonAttributes, 'type'> & {
		variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
		size?: 'md' | 'sm';
		type?: 'button' | 'submit';
		loading?: boolean;
		block?: boolean;
		/** Makes it a link that looks like a button: somewhere to go, not
		    something to do. Resolved by the caller. */
		href?: string;
		target?: HTMLAnchorAttributes['target'];
		rel?: HTMLAnchorAttributes['rel'];
		children: Snippet;
	};

	let {
		variant = 'primary',
		size = 'md',
		type = 'button',
		loading = false,
		block = false,
		href,
		target,
		rel,
		children,
		...rest
	}: Props = $props();

	// The sign-in pages' four kinds of button, drawn as the panel draws them;
	// their regular size is the panel's large one, which its sign-in uses.
	const looks: Record<NonNullable<Props['variant']>, [Variant, ColorPalette]> = {
		primary: ['solid', 'neutral'],
		secondary: ['subtle', 'neutral'],
		danger: ['solid', 'danger'],
		ghost: ['ghost', 'neutral']
	};
	const look = $derived(looks[variant]);
	const drawn = $derived(size === 'md' ? 'lg' : 'sm');
	// Passed on as they came; the shared Button types them where they land.
	const attributes = $derived(rest as Record<string, unknown>);
</script>

{#if href}
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- resolved by the caller -->
	<LinkButton {href} {target} {rel} variant={look[0]} colorPalette={look[1]} size={drawn} {block}>
		{@render children()}
	</LinkButton>
{:else}
	<Button
		{...attributes}
		{type}
		{loading}
		{block}
		variant={look[0]}
		colorPalette={look[1]}
		size={drawn}
	>
		{@render children()}
	</Button>
{/if}
