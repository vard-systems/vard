<script lang="ts">
	import { useTranslator } from '#lib/i18n/index.js';
	import { page } from '$app/state';
	import { Alert, AuthCard } from '#lib/components/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const code = $derived(page.url.searchParams.get('error') ?? 'server_error');

	/** The server's own words, for whoever has to debug. It comes off the URL,
	    so anybody can put anything in it; Svelte escapes it, so it cannot run,
	    and it is capped so it cannot be turned into a page of fake instructions
	    ("call this number…") wearing the provider's address. */
	const description = $derived(
		(page.url.searchParams.get('error_description') ?? '').slice(0, 200)
	);

	const reason = $derived(page.url.searchParams.get('reason') ?? '');
	const said = $derived(reason !== '' && t.has(`error.${reason}`));

	const explanation = $derived.by(() => {
		if (said) return t(`error.${reason}`);

		const key = `error_page.reason.${code}`;
		return t.has(key) ? t(key) : t('error_page.reason.other');
	});
</script>

<svelte:head>
	<title>{t('error_page.head')}</title>
</svelte:head>

<AuthCard organization={data.organization} title={t('error_page.title')} subtitle={explanation}>
	{#if description && !said}
		<Alert tone="warning"><code>{code}</code> {description}</Alert>
	{/if}
	<p class="muted">
		{data.organization?.support_email ? t('error_page.body_support') : t('error_page.body')}
	</p>
</AuthCard>

<style>
	code {
		margin-right: var(--space-1);
		font-family: var(--font-mono);
		font-weight: 700;
	}

	.muted {
		margin-top: var(--space-4);
		color: var(--color-text-hint);
		text-align: center;
	}
</style>
