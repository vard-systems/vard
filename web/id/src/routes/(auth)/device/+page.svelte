<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { ApiError, account } from '#lib/api/index.js';
	import { AuthCard, Decision, ScopeList } from '#lib/components/auth/index.js';
	import { Alert, Button, Form, Note, PageTitle, TextField } from '#lib/components/ui/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	// svelte-ignore state_referenced_locally
	let code = $state(data.code);
	let sending = $state<'yes' | 'no' | null>(null);
	let answered = $state<'allowed' | 'denied' | null>(null);
	let error = $state('');

	const loadError = $derived(
		data.problem ? messageOf(new ApiError(data.problem.status, '', data.problem.code), t) : ''
	);

	/** The code goes in the address, so the page loads what the device asks for. */
	async function look() {
		if (code.trim() === '') return;

		error = '';
		const query = new URLSearchParams({ user_code: code.trim() });
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- resolved, with a query
		await goto(`${resolve('/(auth)/device')}?${query}`, { replace: true });
	}

	async function answer(yes: boolean) {
		error = '';
		sending = yes ? 'yes' : 'no';

		try {
			await account.answerDevice({ user_code: data.code, yes });
			answered = yes ? 'allowed' : 'denied';
		} catch (err) {
			error = messageOf(err, t);
		} finally {
			sending = null;
		}
	}
</script>

<PageTitle parts={[t('device.head')]} />

{#if answered}
	<AuthCard
		organization={data.organization}
		application={data.device?.application}
		title={answered === 'allowed' ? t('device.allowed_title') : t('device.denied_title')}
	>
		<Alert tone={answered === 'allowed' ? 'success' : 'info'}>
			{answered === 'allowed' ? t('device.allowed_body') : t('device.denied_body')}
		</Alert>
	</AuthCard>
{:else if data.device}
	<AuthCard
		organization={data.organization}
		application={data.device.application}
		title={t('device.title')}
		subtitle={t('device.subtitle', { email: data.user.email })}
	>
		{#if error}<Alert>{error}</Alert>{/if}

		<p class="asks">{t('device.asks', { app: data.device.application.name })}</p>
		<ScopeList scopes={data.device.scopes} />

		<Decision yes={t('device.allow')} no={t('device.deny')} {sending} onanswer={answer} />
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		title={t('device.title')}
		subtitle={t('device.subtitle', { email: data.user.email })}
	>
		{#if loadError}<Alert>{loadError}</Alert>{/if}

		<Form onsubmit={look}>
			<Note>{t('device.enter')}</Note>
			<TextField
				label={t('device.code_label')}
				bind:value={code}
				autocomplete="one-time-code"
				autocapitalize="characters"
				spellcheck={false}
				maxlength={16}
				placeholder="BCDF-GHJK"
			/>
			<Button type="submit" block disabled={code.trim() === ''}>{t('device.continue')}</Button>
		</Form>
	</AuthCard>
{/if}

<style>
	.asks {
		margin: 0;
		font-weight: 600;
	}
</style>
