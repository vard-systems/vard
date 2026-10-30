<script lang="ts">
	import { Toaster } from 'ui';
	import { useTranslator } from '#lib/i18n/index.js';

	const t = useTranslator();
</script>

<Toaster label={t('toast.region')} dismiss={t('action.dismiss')} />
