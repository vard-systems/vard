<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApi, ApiError, messageOf } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import GateShell from '#lib/components/layout/GateShell.svelte';
	import { Alert, Button, Input, PasswordInput } from '#lib/components/ui/index.js';

	let { data } = $props();

	/** An authenticator's six digits, which some apps show as "123 456". */
	const CODE_LENGTH = { totp: 7, recovery: 16 } as const;

	let email = $state('');
	let password = $state('');
	let code = $state('');
	let useRecovery = $state(false);
	let error = $state('');
	let submitting = $state(false);
	// svelte-ignore state_referenced_locally
	let step = $state<'password' | 'code'>(data.waitingForCode ? 'code' : 'password');

	const canSubmit = $derived(email.trim() !== '' && password !== '' && !submitting);
	const canVerify = $derived(code.trim().length >= 6 && !submitting);

	async function signIn(event: SubmitEvent) {
		event.preventDefault();
		if (!canSubmit) return;

		error = '';
		submitting = true;

		try {
			const result = await adminApi.login(email.trim(), password);

			if (result.next === 'mfa') {
				step = 'code';
				password = '';
				submitting = false;
				return;
			}

			const target = result.next === 'enroll' ? '/mfa-setup' : '/(panel)';
			await goto(resolve(target), { replace: true, refreshAll: true });
		} catch (err) {
			error = messageOf(err);
			password = '';
			submitting = false;
		}
	}

	async function verify(event: SubmitEvent) {
		event.preventDefault();
		if (!canVerify) return;

		error = '';
		submitting = true;

		try {
			await adminApi.verifyMfa(code.trim());
			await goto(resolve('/(panel)'), { replace: true, refreshAll: true });
		} catch (err) {
			// The half-finished sign-in is gone: only the password starts another.
			if (err instanceof ApiError && err.code === 'admin_sign_in_expired') step = 'password';
			error = messageOf(err);
			code = '';
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
	<title>Sign in · {BRAND.name}</title>
</svelte:head>

{#if step === 'code'}
	<GateShell
		title="Two-factor sign-in"
		description={useRecovery
			? 'Enter one of your recovery codes. Each works once.'
			: 'Enter the six-digit code from your authenticator app.'}
	>
		<form onsubmit={verify}>
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
		<form onsubmit={signIn}>
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
