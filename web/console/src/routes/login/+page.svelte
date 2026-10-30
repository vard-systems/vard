<script lang="ts">
	import { browserSupportsWebAuthn, startAuthentication } from '@simplewebauthn/browser';
	import { onMount } from 'svelte';
	import { enhance } from '$app/forms';
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApi, ApiError, messageOf } from '#lib/api/index.js';
	import { BRAND, pageTitle } from '#lib/brand.js';
	import GateShell from '#lib/components/layout/GateShell.svelte';
	import { Alert, Button, Input, PasswordInput } from '#lib/components/ui/index.js';
	import { submit } from '#lib/forms.js';

	let { data } = $props();

	/** An authenticator's six digits, which some apps show as "123 456". */
	const CODE_LENGTH = { totp: 7, recovery: 16 } as const;

	let email = $state('');
	let password = $state('');
	let code = $state('');
	let useRecovery = $state(false);
	let error = $state('');
	let submitting = $state(false);
	// Known only in the browser; until then the passkey choice is not offered.
	let passkeys = $state(false);

	onMount(() => {
		passkeys = browserSupportsWebAuthn();
	});
	// svelte-ignore state_referenced_locally
	let step = $state<'password' | 'code'>(data.waitingForCode ? 'code' : 'password');

	const canSubmit = $derived(email.trim() !== '' && password !== '' && !submitting);
	const canVerify = $derived(code.trim().length >= 6 && !submitting);

	const signIn = submit<Awaited<ReturnType<typeof adminApi.login>>>({
		input: () => [email.trim(), password],
		when: () => canSubmit,
		pending: (busy) => {
			submitting = busy;
			if (busy) error = '';
		},
		done: async (result) => {
			if (result.next === 'mfa') {
				step = 'code';
				password = '';
				return;
			}

			const target = result.next === 'enroll' ? '/mfa-setup' : '/(panel)';
			await goto(resolve(target), { replace: true, refreshAll: true });
		},
		failed: (err) => {
			error = messageOf(err);
			password = '';
		}
	});

	const verify = submit<unknown>({
		input: () => [code.trim()],
		when: () => canVerify,
		pending: (busy) => {
			submitting = busy;
			if (busy) error = '';
		},
		done: () => goto(resolve('/(panel)'), { replace: true, refreshAll: true }),
		failed: (err) => {
			// The half-finished sign-in is gone: only the password starts another.
			if (err.code === 'admin_sign_in_expired') step = 'password';
			error = messageOf(err);
			code = '';
		}
	});

	async function verifyWithPasskey() {
		error = '';
		submitting = true;

		try {
			const { passkey } = await adminApi.passkeyChallenge();
			const answer = await startAuthentication({ optionsJSON: passkey });
			await adminApi.verifyMfaPasskey(answer);
			await goto(resolve('/(panel)'), { replace: true, refreshAll: true });
		} catch (err) {
			if (err instanceof ApiError && err.code === 'admin_sign_in_expired') step = 'password';
			// Cancelling the browser's dialog is not a mistake worth a message.
			if (!(err instanceof Error && err.name === 'NotAllowedError')) error = messageOf(err);
			submitting = false;
		}
	}

	function switchCode() {
		useRecovery = !useRecovery;
		code = '';
	}

	async function startOver() {
		await adminApi.logout().catch(() => {});
		await refreshAll();
		step = 'password';
		code = '';
		error = '';
	}
</script>

<svelte:head>
	<title>{pageTitle('Sign in')}</title>
</svelte:head>

{#if step === 'code'}
	<GateShell
		title="Two-factor sign-in"
		description={useRecovery
			? 'Enter one of your recovery codes. Each works once.'
			: 'Enter the six-digit code from your authenticator app.'}
	>
		<form method="POST" action="/actions/session?/verifyMfa" use:enhance={verify}>
			{#if error}<Alert>{error}</Alert>{/if}

			{#key useRecovery}
				<Input
					label={useRecovery ? 'Recovery code' : 'Code'}
					bind:value={code}
					inputmode={useRecovery ? 'text' : 'numeric'}
					autocomplete="one-time-code"
					autocapitalize="none"
					spellcheck={false}
					maxlength={useRecovery ? CODE_LENGTH.recovery : CODE_LENGTH.totp}
					disabled={submitting}
				/>
			{/key}

			<Button type="submit" size="lg" loading={submitting} disabled={!canVerify}>
				{submitting ? 'Verifying…' : 'Verify'}
			</Button>

			{#if passkeys}
				<Button variant="subtle" size="lg" disabled={submitting} onclick={verifyWithPasskey}>
					<i class="ri-fingerprint-line" aria-hidden="true"></i>
					Use a passkey
				</Button>
			{/if}

			<div class="links">
				<Button variant="plain" size="sm" onclick={switchCode}>
					{useRecovery ? 'Use the authenticator app' : 'Use a recovery code'}
				</Button>
				<Button variant="plain" size="sm" onclick={startOver}>Start over</Button>
			</div>
		</form>
	</GateShell>
{:else}
	<GateShell title={BRAND.name} description="Sign in to the admin panel">
		<form method="POST" action="/actions/session?/login" use:enhance={signIn}>
			{#if error}<Alert>{error}</Alert>{/if}

			<Input
				label="Email"
				bind:value={email}
				type="email"
				disabled={submitting}
				required
				autocomplete="email"
				autocapitalize="none"
				spellcheck={false}
			/>

			<PasswordInput label="Password" bind:value={password} disabled={submitting} required />

			<Button type="submit" size="lg" loading={submitting} disabled={!canSubmit}>
				{submitting ? 'Signing in…' : 'Sign in'}
			</Button>
		</form>
	</GateShell>
{/if}

<style>
	.links {
		display: flex;
		justify-content: space-between;
		gap: var(--space-2);
	}
</style>
