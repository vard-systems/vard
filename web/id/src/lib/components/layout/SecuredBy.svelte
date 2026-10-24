<script lang="ts">
	import { BRAND } from '#lib/brand.js';
	import { useTranslator } from '#lib/i18n/index.js';

	const t = useTranslator();
</script>

<p class="secured">
	<i class="ri-shield-keyhole-line" style:font-size="15px" aria-hidden="true"></i>
	<span>{t('shell.secured_by')} <strong>{BRAND.name}</strong></span>
</p>

<style>
	.secured {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
