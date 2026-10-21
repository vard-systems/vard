<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { signIn } from '#lib/api/index.js';
	import { Alert, AuthCard, Button } from '#lib/components/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { leaveTo } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);

	let error = $state('');
	let submitting = $state(false);

	// A link anyone could have sent lands here, so signing out waits for the
	// click; the API checks what the link named again and says where to go.
	async function signOut() {
		error = '';
		submitting = true;

		const query = page.url.searchParams;
		try {
			const { location } = await signIn.logout({
				client_id: query.get('client_id') ?? '',
				post_logout_redirect_uri: query.get('post_logout_redirect_uri') ?? '',
				state: query.get('state') ?? ''
			});
			leaveTo(location || resolve('logged-out'));
		} catch (err) {
			error = messageOf(err, t);
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>{t('logout.head')}</title>
</svelte:head>

<AuthCard
	organization={data.organization}
	application={app}
	title={t('logout.title')}
	subtitle={data.user ? t('logout.signed_in_as', { email: data.user.email }) : undefined}
>
	{#if error}<Alert tone="danger">{error}</Alert>{/if}

	<p class="body">{t('logout.body')}</p>

	<div class="actions">
		<Button block loading={submitting} onclick={signOut}>{t('logout.confirm')}</Button>
		{#if app?.website_url}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="secondary" href={app.website_url}>{t('logout.stay', { app: app.name })}</a>
		{:else}
			<a class="secondary" href={resolve('')}>{t('logout.stay_account')}</a>
		{/if}
	</div>
</AuthCard>

<style>
	.body {
		color: var(--color-text-hint);
		text-align: center;
	}

	.actions {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		margin-top: var(--space-5);
	}

	.secondary {
		text-align: center;
	}
</style>
