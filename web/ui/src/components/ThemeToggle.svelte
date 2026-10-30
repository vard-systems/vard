<script lang="ts">
	import type { Theme } from '../theme.svelte';
	import type { Size } from './control';
	import Tooltip from './Tooltip.svelte';

	type Props = {
		/** The app's theme (`new Theme(cookie)`), which this switches. */
		theme: Theme;
		size?: Size;
		variant?: 'ghost' | 'subtle';
		/** What the button is called, and its tooltip: the app's words. */
		label?: string;
		hint?: string;
	};

	let {
		theme,
		size = 'md',
		variant = 'subtle',
		label = 'Switch between the light and dark theme',
		hint = 'Switch theme'
	}: Props = $props();

	const glyph: Record<Size, string> = { sm: '1rem', md: '1.125rem', lg: '1.25rem' };
</script>

<Tooltip label={hint}>
	{#snippet children(trigger)}
		<button
			{...trigger()}
			type="button"
			class="control"
			data-icon="true"
			data-size={size}
			data-variant={variant}
			data-palette="neutral"
			onclick={() => theme.toggle()}
			aria-label={label}
		>
			<span class="icons" style="--glyph: {glyph[size]}">
				<span class="moon"
					><i class="ri-moon-line" style:font-size={glyph[size]} aria-hidden="true"></i></span
				>
				<span class="sun"
					><i class="ri-sun-line" style:font-size={glyph[size]} aria-hidden="true"></i></span
				>
			</span>
		</button>
	{/snippet}
</Tooltip>

<style>
	.icons {
		position: relative;
		display: grid;
		place-items: center;
		width: var(--glyph);
		height: var(--glyph);
	}

	.moon,
	.sun {
		position: absolute;
		display: grid;
		place-items: center;
		transition:
			opacity 200ms ease,
			transform 300ms cubic-bezier(0.4, 0, 0.2, 1);
	}

	.sun {
		opacity: 0;
		transform: rotate(-90deg) scale(0.5);
	}

	:global([data-theme='dark']) .sun {
		opacity: 1;
		transform: rotate(0) scale(1);
	}

	:global([data-theme='dark']) .moon {
		opacity: 0;
		transform: rotate(90deg) scale(0.5);
	}

	@media (prefers-color-scheme: dark) {
		:global(:root:not([data-theme='light'])) .sun {
			opacity: 1;
			transform: rotate(0) scale(1);
		}

		:global(:root:not([data-theme='light'])) .moon {
			opacity: 0;
			transform: rotate(90deg) scale(0.5);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		button,
		.moon,
		.sun {
			transition: none;
		}
	}
</style>
