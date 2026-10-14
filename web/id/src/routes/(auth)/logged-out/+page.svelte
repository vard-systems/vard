<script lang="ts">
	import { useTranslator } from '#lib/i18n/index.js';
	import { resolve } from '$app/paths';
	import { Alert, AuthCard } from '#lib/components/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);
</script>

<svelte:head>
	<title>{t('logged_out.head')}</title>
</svelte:head>

<AuthCard organization={data.organization} application={app} title={t('logged_out.title')}>
	<Alert tone="success">{t('logged_out.body')}</Alert>

	{#if app?.website_url}
		<div class="action">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="primary" href={app.website_url}>{t('logged_out.return', { app: app.name })}</a>
		</div>
	{/if}

	{#snippet below()}
		<a href={resolve('login')}>{t('logged_out.sign_in')}</a>
	{/snippet}
</AuthCard>

<style>
	.action {
		display: flex;
		margin-top: var(--space-5);
	}

	.primary {
		display: flex;
		flex: 1;
		align-items: center;
		justify-content: center;
		height: var(--control-height);
		border-radius: var(--radius-md);
		background: var(--color-primary);
		color: var(--color-primary-text);
		font-weight: 600;
		text-decoration: none;
	}

	.primary:hover {
		background: var(--color-primary-hover);
		text-decoration: none;
	}
</style>
