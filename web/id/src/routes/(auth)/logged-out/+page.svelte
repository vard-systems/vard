<script lang="ts">
	import { resolve } from '$app/paths';
	import { AuthCard } from '#lib/components/auth/index.js';
	import { Alert, Button, PageTitle } from '#lib/components/ui/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);
</script>

<PageTitle parts={[t('logged_out.head')]} />

<AuthCard organization={data.organization} application={app} title={t('logged_out.title')}>
	<Alert tone="success">{t('logged_out.body')}</Alert>

	{#if app?.website_url}
		<Button block href={app.website_url}>{t('logged_out.return', { app: app.name })}</Button>
	{/if}

	{#snippet below()}
		<a href={resolve('login')}>{t('logged_out.sign_in')}</a>
	{/snippet}
</AuthCard>
