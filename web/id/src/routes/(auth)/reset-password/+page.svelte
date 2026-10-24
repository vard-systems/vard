<script lang="ts">
	import { resolve } from '$app/paths';
	import { signIn } from '#lib/api/index.js';
	import { AuthCard } from '#lib/components/auth/index.js';
	import { Alert, Button, Form, NewPasswordFields, PageTitle } from '#lib/components/ui/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { authHref } from '#lib/utils/links.js';
	import { newPasswordReady } from '#lib/utils/password.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);

	let password = $state('');
	let confirm = $state('');
	let error = $state('');
	let submitting = $state(false);
	let done = $state(false);

	const canSubmit = $derived(newPasswordReady(password, confirm) && !submitting);

	async function submit() {
		if (!canSubmit) return;

		error = '';
		submitting = true;

		try {
			await signIn.resetPassword({ token: data.token, password });
			done = true;
		} catch (err) {
			error = messageOf(err, t);
		} finally {
			submitting = false;
		}
	}
</script>

<PageTitle parts={[t('reset.head'), app?.name]} />

{#if done}
	<AuthCard organization={data.organization} application={app} title={t('reset.done_title')}>
		<Alert tone="success">{t('reset.done_body')}</Alert>

		<Button block href={authHref('/login', data.openRequest)}>
			{app ? t('reset.done_action_app', { app: app.name }) : t('reset.done_action')}
		</Button>
	</AuthCard>
{:else if !data.valid}
	<AuthCard organization={data.organization} application={app} title={t('reset.expired_title')}>
		<Alert tone="info">{t('reset.expired_body')}</Alert>

		{#snippet below()}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/forgot-password', data.openRequest)}>{t('reset.expired_new')}</a>
			·
			<a href={resolve('login')}>{t('action.sign_in')}</a>
		{/snippet}
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('reset.title')}
		subtitle={data.temporary ? t('reset.temporary') : undefined}
	>
		<Form onsubmit={submit}>
			{#if error}<Alert>{error}</Alert>{/if}

			<NewPasswordFields bind:password bind:confirm disabled={submitting} />

			<Button type="submit" block loading={submitting} disabled={!canSubmit}>
				{submitting ? t('reset.submitting') : t('reset.submit')}
			</Button>
		</Form>
	</AuthCard>
{/if}
