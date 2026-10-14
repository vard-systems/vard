<script lang="ts">
	import { page } from '$app/state';
	import { signIn } from '#lib/api/index.js';
	import { Alert, AuthCard, Button } from '#lib/components/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { authHref } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.signInRequest?.application ?? null);
	const request = $derived(data.signInRequest ? data.request : null);
	const token = $derived(page.url.searchParams.get('token') ?? '');

	let error = $state('');
	let submitting = $state(false);
	let done = $state(false);

	// The link only opens this page; confirming is a button press. A mail
	// scanner that opens every link in an inbox would otherwise use it up
	// before its owner ever saw it.
	async function confirm() {
		error = '';
		submitting = true;

		try {
			await signIn.verifyEmail({ token });
			done = true;
		} catch (err) {
			error = messageOf(err, t);
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>{t('verify.head')}{app ? ` · ${app.name}` : ''}</title>
</svelte:head>

{#if done}
	<AuthCard organization={data.organization} application={app} title={t('verify.done_title')}>
		<Alert tone="success">{t('verify.done_body')}</Alert>

		<div class="action">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="primary" href={authHref('/login', request)}>
				{app ? t('verify.continue_app', { app: app.name }) : t('verify.continue')}
			</a>
		</div>
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('verify.title')}
		subtitle={t('verify.subtitle')}
	>
		<div class="body">
			{#if !token}
				<Alert tone="info">{t('error.verification_invalid')}</Alert>
			{:else}
				{#if error}<Alert>{error}</Alert>{/if}

				<Button block loading={submitting} onclick={confirm}>
					{submitting ? t('verify.submitting') : t('verify.submit')}
				</Button>
			{/if}
		</div>
	</AuthCard>
{/if}

<style>
	.body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

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
