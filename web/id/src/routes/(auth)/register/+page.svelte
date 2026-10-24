<script lang="ts">
	import { ApiError, requiredSSO, signIn } from '#lib/api/index.js';
	import { AuthCard, SocialButtons } from '#lib/components/auth/index.js';
	import {
		Alert,
		Button,
		Checkbox,
		FieldRow,
		Form,
		NewPasswordFields,
		PageTitle,
		TextField
	} from '#lib/components/ui/index.js';
	import { LIMITS } from '#lib/constants.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { legalLinks } from '#lib/utils/legal.js';
	import { authHref, leaveTo, ssoHref } from '#lib/utils/links.js';
	import { newPasswordReady } from '#lib/utils/password.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);

	const agreements = $derived(legalLinks(app, data.organization));
	const needsConsent = $derived(agreements.length > 0);

	let firstName = $state('');
	let lastName = $state('');
	// svelte-ignore state_referenced_locally
	let email = $state(data.signInRequest?.login_hint ?? '');
	let password = $state('');
	let confirm = $state('');
	let accepted = $state(false);
	let error = $state('');
	let sent = $state('');
	let submitting = $state(false);

	const canSubmit = $derived(
		email.trim() !== '' &&
			newPasswordReady(password, confirm) &&
			(!needsConsent || accepted) &&
			!submitting
	);

	async function submit() {
		if (!canSubmit || !data.request) return;

		error = '';
		submitting = true;

		try {
			const result = await signIn.register({
				request: data.request,
				email: email.trim(),
				password,
				first_name: firstName.trim(),
				last_name: lastName.trim(),
				accept_terms: accepted,
				remember: data.login.allow_remember_me
			});

			if (result.redirect_to) {
				leaveTo(result.redirect_to);
				return;
			}

			submitting = false;
		} catch (err) {
			const sso = requiredSSO(err);
			if (sso) {
				leaveTo(ssoHref(sso.slug, { request: data.request, email: email.trim() }));
				return;
			}

			if (err instanceof ApiError && err.code === 'email_not_verified') {
				sent = messageOf(err, t);
			} else {
				error = messageOf(err, t);
			}
			submitting = false;
		}
	}
</script>

<PageTitle parts={[t('register.head'), app?.name]} />

{#if sent}
	<AuthCard organization={data.organization} application={app} title={t('verify.title')}>
		<Alert tone="info">{sent}</Alert>

		{#snippet below()}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/login', data.request)}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{:else if data.expired || !data.request}
	<AuthCard
		organization={data.organization}
		title={data.expired ? t('login.expired_title') : t('register.start_title')}
		subtitle={t('register.start_subtitle')}
	>
		<Alert tone="info">{t('login.expired_body')}</Alert>
	</AuthCard>
{:else if app && !app.allow_registration}
	<AuthCard organization={data.organization} application={app} title={t('register.closed_title')}>
		<Alert tone="info">{t('register.closed_body', { app: app.name })}</Alert>

		{#snippet below()}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/login', data.request)}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('register.title')}
		subtitle={app ? t('register.subtitle_app', { app: app.name }) : undefined}
	>
		<Form onsubmit={submit}>
			{#if error}<Alert>{error}</Alert>{/if}

			<FieldRow>
				<TextField
					label={t('field.first_name')}
					bind:value={firstName}
					autocomplete="given-name"
					maxlength={LIMITS.name}
					disabled={submitting}
				/>
				<TextField
					label={t('field.last_name')}
					bind:value={lastName}
					autocomplete="family-name"
					maxlength={LIMITS.name}
					disabled={submitting}
				/>
			</FieldRow>

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

			<NewPasswordFields bind:password bind:confirm initial disabled={submitting} />

			{#if needsConsent}
				<Checkbox bind:checked={accepted} disabled={submitting}>
					{t('register.consent')}
					{#each agreements as agreement, index (agreement.key)}
						{#if index > 0}{t('register.consent_and')}{/if}
						<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
						<a href={agreement.href} target="_blank" rel="noopener noreferrer">
							{t(agreement.key)}
						</a>
					{/each}
				</Checkbox>
			{/if}

			<Button type="submit" block loading={submitting} disabled={!canSubmit}>
				{submitting ? t('register.submitting') : t('register.submit')}
			</Button>
		</Form>

		<SocialButtons
			providers={data.socialProviders}
			request={data.request}
			label={t('register.social')}
			disabled={submitting}
		/>

		{#snippet below()}
			{t('register.have_account')}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/login', data.request)}>{t('action.sign_in')}</a>
		{/snippet}
	</AuthCard>
{/if}
