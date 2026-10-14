<script lang="ts">
	import { RiMoonLine, RiSunLine } from 'svelte-remixicon';
	import { THEME_KEY } from '#lib/brand.js';
	import Icon from './Icon.svelte';

	/** What the page shows now: the reader's choice, or the system's. */
	function isDark(): boolean {
		const chosen = document.documentElement.dataset.theme;
		return chosen ? chosen === 'dark' : matchMedia('(prefers-color-scheme: dark)').matches;
	}

	function apply(theme: 'dark' | 'light') {
		document.documentElement.dataset.theme = theme;
		try {
			localStorage.setItem(THEME_KEY, theme);
		} catch {
			// A private window may refuse; the choice lasts for this page.
		}
	}

	// One cross-fade of the whole page, as the console switches.
	function toggle() {
		const next = isDark() ? 'light' : 'dark';
		if (document.startViewTransition) document.startViewTransition(() => apply(next));
		else apply(next);
	}
</script>

<!-- Both icons are drawn, and CSS shows the one for the theme you would
     switch to, keyed on the page's theme: right in the first frame, with no
     guess to correct after the script runs. As the console's toggle. -->
<button
	type="button"
	class="control"
	data-tooltip="Switch theme"
	data-tooltip-end
	aria-label="Switch between the light and dark theme"
	onclick={toggle}
>
	<span class="icons">
		<span class="moon"><Icon icon={RiMoonLine} /></span>
		<span class="sun"><Icon icon={RiSunLine} /></span>
	</span>
</button>

<style>
	.icons {
		position: relative;
		display: grid;
		place-items: center;
		width: 1rem;
		height: 1rem;
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
		transform: none;
	}

	:global([data-theme='dark']) .moon {
		opacity: 0;
		transform: rotate(90deg) scale(0.5);
	}

	@media (prefers-color-scheme: dark) {
		:global(:root:not([data-theme='light'])) .sun {
			opacity: 1;
			transform: none;
		}

		:global(:root:not([data-theme='light'])) .moon {
			opacity: 0;
			transform: rotate(90deg) scale(0.5);
		}
	}
</style>
