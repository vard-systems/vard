<script lang="ts">
	import { resolve } from '$app/paths';
	import { signIn } from '#lib/api/index.js';
	import { Alert, AuthCard, Button, PasswordField } from '#lib/components/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { authHref } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.signInRequest?.application ?? null);
	const request = $derived(data.signInRequest ? data.request : null);

	const minLength = 8;

	let password = $state('');
	let confirm = $state('');
	let error = $state('');
	let submitting = $state(false);
	let done = $state(false);

	const mismatch = $derived(confirm !== '' && confirm !== password);
	const canSubmit = $derived(password.length >= minLength && confirm === password && !submitting);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
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

<svelte:head>
	<title>{t('reset.head')}{app ? ` · ${app.name}` : ''}</title>
</svelte:head>

{#if done}
	<AuthCard organization={data.organization} application={app} title={t('reset.done_title')}>
		<Alert tone="success">{t('reset.done_body')}</Alert>

		<div class="action">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="primary" href={authHref('/login', request)}>
				{app ? t('reset.done_action_app', { app: app.name }) : t('reset.done_action')}
			</a>
		</div>
	</AuthCard>
{:else if !data.valid}
	<AuthCard organization={data.organization} application={app} title={t('reset.expired_title')}>
		<Alert tone="info">{t('reset.expired_body')}</Alert>

		{#snippet below()}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={authHref('/forgot-password', request)}>{t('reset.expired_new')}</a>
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
		<form onsubmit={submit} novalidate>
			{#if error}<Alert>{error}</Alert>{/if}

			<PasswordField
				label={t('field.new_password')}
				bind:value={password}
				autocomplete="new-password"
				disabled={submitting}
				hint={t('field.password_hint', { count: minLength })}
			/>

			<PasswordField
				label={t('field.confirm_new_password')}
				bind:value={confirm}
				autocomplete="new-password"
				disabled={submitting}
				error={mismatch ? t('field.password_mismatch') : undefined}
			/>

			<Button type="submit" block loading={submitting} disabled={!canSubmit}>
				{submitting ? t('reset.submitting') : t('reset.submit')}
			</Button>
		</form>
	</AuthCard>
{/if}

<style>
	form {
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
