<script lang="ts">
	import { ThemeToggle } from 'ui';
	import { useTranslator } from '#lib/i18n/index.js';
	import { theme } from '#lib/state/theme.svelte.js';

	const t = useTranslator();
</script>

<ThemeToggle {theme} label={t('shell.theme')} hint={t('shell.theme')} />
