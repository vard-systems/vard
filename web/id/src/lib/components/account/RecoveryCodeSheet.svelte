<script lang="ts">
	import { BRAND } from '#lib/brand.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import { saveText } from '#lib/utils/download.js';
	import { Alert, Button } from '../ui/index.js';

	type Props = { codes: string[]; ondone: () => void };

	let { codes, ondone }: Props = $props();

	const t = useTranslator();
</script>

<div class="sheet">
	<Alert tone="warning">{t('security.recovery_save')}</Alert>

	<ol class="codes">
		{#each codes as code (code)}
			<li><code>{code}</code></li>
		{/each}
	</ol>

	<div class="actions">
		<Button
			variant="secondary"
			size="sm"
			onclick={() => saveText(`${BRAND.slug}-recovery-codes.txt`, codes.join('\n') + '\n')}
		>
			<i class="ri-download-line" aria-hidden="true"></i>
			{t('security.recovery_download')}
		</Button>
		<Button size="sm" onclick={ondone}>{t('security.recovery_done')}</Button>
	</div>
</div>

<style>
	.sheet {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.codes {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
		gap: var(--space-2);
		margin: 0;
		padding: var(--space-3);
		border: 1px dashed var(--color-border-strong);
		border-radius: var(--radius-md);
		list-style: none;
	}

	code {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		letter-spacing: 0.04em;
	}

	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
