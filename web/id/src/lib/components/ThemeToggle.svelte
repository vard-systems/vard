<script lang="ts">
	import { COOKIES } from '#lib/brand.js';
	import { useTranslator } from '#lib/i18n/index.js';

	const t = useTranslator();

	function toggle() {
		const root = document.documentElement;
		const current =
			root.dataset.theme ?? (matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
		const next = current === 'dark' ? 'light' : 'dark';

		root.dataset.theme = next;
		root.style.colorScheme = next;
		document.cookie = `${COOKIES.theme}=${next}; path=/; max-age=31536000; samesite=lax`;
	}
</script>

<button type="button" class="toggle" onclick={toggle} aria-label={t('shell.theme')}>
	<span class="moon"><i class="ri-moon-line" aria-hidden="true"></i></span>
	<span class="sun"><i class="ri-sun-line" aria-hidden="true"></i></span>
</button>

<style>
	.toggle {
		display: grid;
		place-items: center;
		width: 38px;
		height: 38px;
		border: 0;
		border-radius: var(--radius-md);
		background: transparent;
		color: var(--color-text-hint);
		cursor: pointer;
	}

	.toggle:hover {
		background: var(--color-input);
		color: var(--color-text);
	}

	.moon,
	.sun {
		display: grid;
	}

	.sun {
		display: none;
	}

	:global(:root[data-theme='dark']) .moon {
		display: none;
	}

	:global(:root[data-theme='dark']) .sun {
		display: grid;
	}

	@media (prefers-color-scheme: dark) {
		:global(:root:not([data-theme='light'])) .moon {
			display: none;
		}

		:global(:root:not([data-theme='light'])) .sun {
			display: grid;
		}
	}
</style>
