<script lang="ts">
	import { requiredSSO, signIn } from '$lib/api';
	import { Alert, AuthCard, Button, TextField } from '$lib/components';
	import { messageOf, useTranslator } from '$lib/i18n';
	import { authHref, leaveTo, ssoHref } from '$lib/utils/links';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.signInRequest?.application ?? null);
	const request = $derived(data.signInRequest ? data.request : null);

	// svelte-ignore state_referenced_locally
	let email = $state(data.signInRequest?.login_hint ?? '');
	let error = $state('');
	let submitting = $state(false);
	let sentTo = $state('');

	const canSubmit = $derived(email.trim() !== '' && !submitting);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!canSubmit) return;

		error = '';
		submitting = true;

		try {
			await signIn.forgotPassword({
				request: request ?? '',
				email: email.trim(),
				language: data.language
			});
			sentTo = email.trim();
		} catch (err) {
			const sso = requiredSSO(err);
			if (sso) {
				leaveTo(ssoHref(sso.slug, { request: data.request, email: email.trim() }));
				return;
			}

			error = messageOf(err, t);
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>{t('forgot.head')}{app ? ` · ${app.name}` : ''}</title>
</svelte:head>

{#if sentTo}
	<AuthCard organization={data.organization} application={app} title={t('forgot.sent_title')}>
		<!-- The same words whether or not the address has an account, so the
		     page cannot be used to find out which ones do. -->
		<Alert tone="success">{t('forgot.sent_body', { email: sentTo })}</Alert>
		<p class="muted">{t('forgot.sent_hint')}</p>

		{#snippet below()}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/login', request)}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('forgot.title')}
		subtitle={t('forgot.subtitle')}
	>
		<form onsubmit={submit} novalidate>
			{#if error}<Alert>{error}</Alert>{/if}

			<TextField
				label={t('field.email')}
				bind:value={email}
				type="email"
				autocomplete="email"
				autocapitalize="none"
				spellcheck={false}
				disabled={submitting}
			/>

			<Button type="submit" block loading={submitting} disabled={!canSubmit}>
				{submitting ? t('forgot.submitting') : t('forgot.submit')}
			</Button>
		</form>

		{#snippet below()}
			{t('forgot.remembered')}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/login', request)}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{/if}

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.muted {
		margin-top: var(--space-4);
		color: var(--color-text-hint);
		text-align: center;
	}
</style>
