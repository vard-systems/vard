<script lang="ts">
	import { requiredSSO, signIn } from '#lib/api/index.js';
	import { AuthCard } from '#lib/components/auth/index.js';
	import { Alert, Button, Form, Note, PageTitle, TextField } from '#lib/components/ui/index.js';
	import { LIMITS } from '#lib/constants.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { authHref, leaveTo, ssoHref } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);

	// svelte-ignore state_referenced_locally
	let email = $state(data.signInRequest?.login_hint ?? '');
	let error = $state('');
	let submitting = $state(false);
	let sentTo = $state('');

	const canSubmit = $derived(email.trim() !== '' && !submitting);

	async function submit() {
		if (!canSubmit) return;

		error = '';
		submitting = true;

		try {
			await signIn.forgotPassword({
				request: data.openRequest ?? '',
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

<PageTitle parts={[t('forgot.head'), app?.name]} />

{#if sentTo}
	<AuthCard organization={data.organization} application={app} title={t('forgot.sent_title')}>
		<!-- The same words whether or not the address has an account, so the
		     page cannot be used to find out which ones do. -->
		<Alert tone="success">{t('forgot.sent_body', { email: sentTo })}</Alert>
		<Note align="center">{t('forgot.sent_hint')}</Note>

		{#snippet below()}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/login', data.openRequest)}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('forgot.title')}
		subtitle={t('forgot.subtitle')}
	>
		<Form onsubmit={submit}>
			{#if error}<Alert>{error}</Alert>{/if}

			<TextField
				label={t('field.email')}
				bind:value={email}
				type="email"
				autocomplete="email"
				autocapitalize="none"
				spellcheck={false}
				maxlength={LIMITS.email}
				disabled={submitting}
			/>

			<Button type="submit" block loading={submitting} disabled={!canSubmit}>
				{submitting ? t('forgot.submitting') : t('forgot.submit')}
			</Button>
		</Form>

		{#snippet below()}
			{t('forgot.remembered')}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/login', data.openRequest)}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{/if}
