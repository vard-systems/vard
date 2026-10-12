<script lang="ts">
	import { resolve } from '$app/paths';
	import { signIn } from '$lib/api';
	import { Alert, AuthCard, Button, TextField } from '$lib/components';
	import { messageOf, useTranslator } from '$lib/i18n';
	import { authHref, leaveTo, ssoHref } from '$lib/utils/links';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.signInRequest?.application ?? null);

	// svelte-ignore state_referenced_locally
	let email = $state(data.signInRequest?.login_hint ?? '');
	let error = $state('');
	let submitting = $state(false);
	let redirecting = $state('');

	const canSubmit = $derived(email.trim().includes('@') && !submitting);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!canSubmit) return;

		error = '';
		submitting = true;

		try {
			const { connection } = await signIn.discoverSSO(email.trim());
			redirecting = connection.name;
			leaveTo(
				ssoHref(connection.slug, { request: data.request, next: data.next, email: email.trim() })
			);
		} catch (err) {
			error = messageOf(err, t);
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>{t('login.sso_title')}{app ? ` · ${app.name}` : ''}</title>
</svelte:head>

<AuthCard
	organization={data.organization}
	application={app}
	title={t('login.sso_title')}
	subtitle={t('login.sso_subtitle')}
>
	<form onsubmit={submit} novalidate>
		{#if error}<Alert>{error}</Alert>{/if}
		{#if redirecting}
			<Alert tone="info">{t('login.sso_redirecting', { name: redirecting })}</Alert>
		{/if}

		<TextField
			label={t('field.email')}
			bind:value={email}
			type="email"
			name="email"
			autocomplete="username"
			autocapitalize="none"
			spellcheck={false}
			disabled={submitting}
		/>

		<Button type="submit" block loading={submitting} disabled={!canSubmit}>
			{t('login.sso_submit')}
		</Button>
	</form>

	{#snippet below()}
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		<a href={data.request ? authHref('/login', data.request) : resolve('/login')}>
			{t('login.use_password')}
		</a>
	{/snippet}
</AuthCard>

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}
</style>
