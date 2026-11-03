<script lang="ts">
	import { page } from '$app/state';
	import { ApiError, signIn } from '#lib/api/index.js';
	import { AuthCard } from '#lib/components/auth/index.js';
	import { Alert, Button, Form, PageTitle, PasswordField } from '#lib/components/ui/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { authHref } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);
	const token = $derived(page.url.searchParams.get('token') ?? '');

	let error = $state('');
	let submitting = $state(false);
	let done = $state(false);
	// What the server asks for before it takes the link: the account's
	// password, or a browser signed in to it.
	let asks = $state<'' | 'password' | 'sign_in'>('');
	let password = $state('');

	// The link only opens this page; confirming is a button press. A mail
	// scanner that opens every link in an inbox would otherwise use it up
	// before its owner ever saw it.
	async function confirm() {
		error = '';
		submitting = true;

		try {
			await signIn.verifyEmail(asks === 'password' ? { token, password } : { token });
			done = true;
		} catch (err) {
			if (err instanceof ApiError && err.code === 'verification_password_required') {
				asks = 'password';
			} else if (err instanceof ApiError && err.code === 'verification_sign_in_required') {
				asks = 'sign_in';
			} else {
				error = messageOf(err, t);
			}
			password = '';
		} finally {
			submitting = false;
		}
	}
</script>

<PageTitle parts={[t('verify.head'), app?.name]} />

{#if done}
	<AuthCard organization={data.organization} application={app} title={t('verify.done_title')}>
		<Alert tone="success">{t('verify.done_body')}</Alert>

		<Button block href={authHref('/login', data.openRequest)}>
			{app ? t('verify.continue_app', { app: app.name }) : t('verify.continue')}
		</Button>
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('verify.title')}
		subtitle={t('verify.subtitle')}
	>
		{#if !token}
			<Alert tone="info">{t('error.verification_invalid')}</Alert>
		{:else if asks === 'sign_in'}
			<Alert tone="info">{t('error.verification_sign_in_required')}</Alert>
			<Button block href={authHref('/login', data.openRequest)}>{t('verify.continue')}</Button>
		{:else if asks === 'password'}
			<Form onsubmit={confirm}>
				{#if error}<Alert>{error}</Alert>{:else}<Alert tone="info"
						>{t('error.verification_password_required')}</Alert
					>{/if}

				<PasswordField
					label={t('field.password')}
					bind:value={password}
					hint={t('verify.password_hint')}
					disabled={submitting}
					aside={{
						label: t('verify.set_password'),
						href: authHref('/forgot-password', data.openRequest)
					}}
				/>

				<Button type="submit" block loading={submitting} disabled={password === '' || submitting}>
					{submitting ? t('verify.submitting') : t('verify.submit')}
				</Button>
			</Form>
		{:else}
			{#if error}<Alert>{error}</Alert>{/if}

			<Button block loading={submitting} onclick={confirm}>
				{submitting ? t('verify.submitting') : t('verify.submit')}
			</Button>
		{/if}
	</AuthCard>
{/if}
