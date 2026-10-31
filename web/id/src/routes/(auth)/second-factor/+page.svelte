<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { browserSupportsWebAuthn, startAuthentication } from '@simplewebauthn/browser';
	import { signIn, type SecondFactorAnswer, type SignedIn } from '#lib/api/index.js';
	import { AuthCard } from '#lib/components/auth/index.js';
	import { Alert, Button, Form, Note, PageTitle, TextField } from '#lib/components/ui/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { authHref, leaveTo } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);
	const challenge = $derived(data.challenge);
	const offers = $derived({
		authenticator: challenge?.methods.includes('totp') ?? false,
		passkey: (challenge?.methods.includes('passkey') ?? false) && challenge?.passkey !== undefined
	});

	/** Typing a recovery code instead: the default when there is no authenticator. */
	// svelte-ignore state_referenced_locally
	let recovering = $state(!offers.authenticator);
	let code = $state('');
	let error = $state('');
	let sending = $state<'code' | 'passkey' | null>(null);

	async function arrive(result: SignedIn) {
		if (result.redirect_to) {
			leaveTo(result.redirect_to);
			return;
		}

		// eslint-disable-next-line svelte/no-navigation-without-resolve -- a checked path on this site
		await goto(data.next, { refreshAll: true, replace: true });
	}

	async function send(answer: SecondFactorAnswer, kind: 'code' | 'passkey') {
		error = '';
		sending = kind;

		try {
			await arrive(await signIn.submitSecondFactor(answer));
		} catch (err) {
			error = messageOf(err, t);
			code = '';
			sending = null;
			// A passkey's challenge is spent on its first answer; the next
			// try needs the fresh one the page's load asks for.
			if (kind === 'passkey') await refreshAll();
		}
	}

	/** A new kind of code starts clean: the last one's mistake is not about it. */
	function switchMode() {
		recovering = !recovering;
		code = '';
		error = '';
	}

	async function usePasskey() {
		if (!challenge?.passkey || !browserSupportsWebAuthn()) return;

		error = '';
		let response;
		try {
			response = await startAuthentication({ optionsJSON: challenge.passkey });
		} catch {
			error = t('second_factor.passkey_failed');
			return;
		}

		await send({ passkey: response }, 'passkey');
	}
</script>

<PageTitle parts={[t('second_factor.head'), app?.name]} />

{#if !challenge}
	<AuthCard organization={data.organization} title={t('second_factor.expired_title')}>
		<Alert tone="info">{t('second_factor.expired_body')}</Alert>

		{#snippet below()}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- authHref resolves it -->
			<a href={authHref('/login', data.openRequest)}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('second_factor.title')}
		subtitle={app ? t('second_factor.subtitle_app', { app: app.name }) : undefined}
	>
		{#if error}<Alert tone="danger">{error}</Alert>{/if}

		{#if offers.passkey}
			<Button
				block
				loading={sending === 'passkey'}
				disabled={sending !== null}
				onclick={usePasskey}
			>
				<i class="ri-fingerprint-line" aria-hidden="true"></i>
				{t('second_factor.passkey')}
			</Button>
			<p class="or"><span>{t('second_factor.or')}</span></p>
		{/if}

		<Form onsubmit={() => send({ code }, 'code')}>
			{#if recovering}
				<TextField
					label={t('second_factor.recovery_label')}
					hint={t('second_factor.recovery_hint')}
					bind:value={code}
					autocomplete="off"
					spellcheck={false}
					maxlength={32}
					disabled={sending !== null}
				/>
			{:else}
				<TextField
					label={t('second_factor.code_label')}
					hint={t('second_factor.code_hint')}
					bind:value={code}
					inputmode="numeric"
					autocomplete="one-time-code"
					maxlength={8}
					disabled={sending !== null}
				/>
			{/if}

			<Button
				type="submit"
				block
				variant={offers.passkey ? 'secondary' : 'primary'}
				loading={sending === 'code'}
				disabled={code.trim() === '' || sending !== null}
			>
				{t('second_factor.submit')}
			</Button>
		</Form>

		{#if offers.authenticator}
			<Note size="sm" align="center">
				<button type="button" class="switch" onclick={switchMode}>
					{recovering ? t('second_factor.use_authenticator') : t('second_factor.use_recovery')}
				</button>
			</Note>
		{/if}

		{#snippet below()}
			<a href={resolve('login')}>{t('action.back_to_sign_in')}</a>
		{/snippet}
	</AuthCard>
{/if}

<style>
	.or {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.or::before,
	.or::after {
		flex: 1;
		height: 1px;
		background: var(--color-border);
		content: '';
	}

	.switch {
		padding: 0;
		border: 0;
		background: none;
		color: var(--color-link);
		font: inherit;
		cursor: pointer;
	}

	.switch:hover {
		text-decoration: underline;
	}
</style>
