<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import {
		ApiError,
		requiredSSO,
		signIn,
		type CodeChallenge,
		type SignedIn
	} from '#lib/api/index.js';

	import {
		Alert,
		AuthCard,
		Button,
		Checkbox,
		CodeForm,
		PasswordField,
		TextField,
		SocialButtons,
		SSOButtons
	} from '#lib/components/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { authHref, leaveTo, ssoHref } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.signInRequest?.application ?? null);

	const flow = $derived(data.login);
	const offerSocial = $derived(flow.steps.includes('social'));
	const offerPassword = $derived(flow.steps.includes('password'));
	const offerRegistration = $derived(
		offerPassword && flow.allow_registration && app?.allow_registration === true
	);
	const closed = $derived(!flow.allow_sign_in);
	const offerRemember = $derived(flow.allow_remember_me);

	// The application may already know who is signing in (login_hint).
	// svelte-ignore state_referenced_locally
	let email = $state(data.signInRequest?.login_hint ?? '');
	let password = $state('');
	/** "Stay signed in": off by default, because the machine somebody is
	    signing in on is not known to be theirs. */
	let remember = $state(false);
	let error = $state('');
	let notice = $state('');
	let submitting = $state(false);
	let redirecting = $state('');
	let challenge = $state<CodeChallenge | null>(null);

	const canSubmit = $derived(email.trim() !== '' && password !== '' && !submitting);

	async function arrive(result: SignedIn, next: string) {
		if (result.password_change_required && result.reset_token) {
			let query = `token=${encodeURIComponent(result.reset_token)}&reason=temporary`;
			if (data.request) query += `&request=${encodeURIComponent(data.request)}`;
			// replace so the reset token does not stay in the browser's
			// history: this entry replaces the sign-in page rather than stacking
			// on it, and Back does not return to a URL carrying a live token.
			// eslint-disable-next-line svelte/no-navigation-without-resolve -- resolved, with a query
			await goto(`${resolve('reset-password')}?${query}`, { replace: true });
			return;
		}

		if (result.code) {
			challenge = result.code;
			password = '';
			submitting = false;
			return;
		}

		if (result.redirect_to) {
			leaveTo(result.redirect_to);
			return;
		}

		// Signed in to the account itself. One navigation, which reloads the
		// data on the way: a separate refreshAll would re-run this page's
		// load, whose redirect for a signed-in user races this goto. `next` is
		// a checked path on this site, but one with no route makes goto reject,
		// so it is loaded the plain way and shows the not-found page.
		try {
			// eslint-disable-next-line svelte/no-navigation-without-resolve -- a checked path on this site
			await goto(next, { refreshAll: true, replace: true });
		} catch {
			window.location.replace(next);
		}
	}

	function signedInWithCode(result: SignedIn) {
		void arrive(result, data.next);
	}

	function codeExpired(message: string) {
		challenge = null;
		submitting = false;
		password = '';
		error = message;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!canSubmit) return;

		error = '';
		notice = '';
		submitting = true;

		// Read before anything is awaited: once the session changes, this
		// page's data can be replaced under it.
		const next = data.next;

		try {
			const result = await signIn.login({
				request: data.request ?? '',
				email: email.trim(),
				password,
				remember
			});

			await arrive(result, next);
		} catch (err) {
			const sso = requiredSSO(err);
			if (sso) {
				redirecting = sso.name;
				password = '';
				leaveTo(ssoHref(sso.slug, { request: data.request, next, email: email.trim() }));
				return;
			}

			if (err instanceof ApiError && err.code === 'email_not_verified') {
				notice = messageOf(err, t);
			} else {
				error = messageOf(err, t);
			}
			password = '';
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>{t('login.head')}{app ? ` · ${app.name}` : ''}</title>
</svelte:head>

{#if data.expired}
	<AuthCard
		organization={data.organization}
		title={t('login.expired_title')}
		subtitle={t('login.expired_subtitle')}
	>
		<Alert tone="info">{t('login.expired_body')}</Alert>

		{#snippet below()}
			{t('login.expired_or')}
			<a href={resolve('login')}>{t('login.expired_link')}</a>
		{/snippet}
	</AuthCard>
{:else if closed}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('login.closed_title')}
		subtitle={app ? t('login.subtitle_app', { app: app.name }) : t('login.subtitle_account')}
	>
		<Alert tone="warning">{t('login.closed_body')}</Alert>
	</AuthCard>
{:else if challenge}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('login.code_title')}
		subtitle={t('login.code_subtitle', { email: challenge.email })}
	>
		<CodeForm {challenge} onsignedin={signedInWithCode} onexpired={codeExpired} />

		{#snippet below()}
			<a href={resolve('login')}>{t('login.code_back')}</a>
		{/snippet}
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('login.title')}
		subtitle={app ? t('login.subtitle_app', { app: app.name }) : t('login.subtitle_account')}
	>
		{#if offerPassword}
			<form onsubmit={submit} novalidate>
				{#if error}<Alert>{error}</Alert>{/if}
				{#if notice}<Alert tone="info">{notice}</Alert>{/if}
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

				<PasswordField
					label={t('field.password')}
					bind:value={password}
					disabled={submitting}
					aside={flow.allow_password_reset
						? { label: t('login.forgot'), href: authHref('/forgot-password', data.request) }
						: undefined}
				/>

				{#if offerRemember}
					<Checkbox bind:checked={remember} disabled={submitting}>
						{t('login.remember')}
						<span class="aside">{t('login.remember_hint')}</span>
					</Checkbox>
				{/if}

				<Button type="submit" block loading={submitting} disabled={!canSubmit}>
					{submitting ? t('login.submitting') : t('login.submit')}
				</Button>
			</form>
		{/if}

		<SocialButtons
			providers={offerSocial ? data.socialProviders : []}
			request={data.request}
			next={data.next ?? null}
			label={t('login.social')}
			disabled={submitting}
		/>

		<SSOButtons
			connections={data.ssoConnections}
			request={data.request}
			next={data.next ?? null}
			disabled={submitting}
		/>

		{#if data.ssoAvailable}
			<p class="sso-link">
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={authHref('/sso', data.request)}>{t('login.sso')}</a>
			</p>
		{/if}

		{#snippet below()}
			{#if offerRegistration && data.request}
				{t('login.no_account')}
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={authHref('/register', data.request)}>{t('login.create_one')}</a>
			{/if}
		{/snippet}
	</AuthCard>
{/if}

<style>
	.sso-link {
		margin: var(--space-4) 0 0;
		text-align: center;
		font-size: var(--text-sm);
	}

	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.aside {
		display: block;
		color: var(--color-text-muted);
		font-size: var(--text-sm);
	}
</style>
