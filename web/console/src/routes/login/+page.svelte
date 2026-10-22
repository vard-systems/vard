<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApi, ApiError } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		Alert,
		Button,
		Card,
		Input,
		PasswordInput,
		ThemeToggle
	} from '#lib/components/ui/index.js';

	let { data } = $props();

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
			error = err instanceof ApiError ? err.message : 'Something went wrong';
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
			if (err instanceof ApiError && err.message.includes('expired')) step = 'password';
			error = err instanceof ApiError ? err.message : 'Something went wrong';
			code = '';
			submitting = false;
		}
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

<main>
	<div class="corner">
		<ThemeToggle />
	</div>

	<div class="panel">
		<Card padded>
			{#if step === 'code'}
				<form onsubmit={verify}>
					<header>
						<span class="mark">
							<i
								class="ri-shield-keyhole-line"
								style:font-size="20px"
								role="img"
								aria-label={BRAND.name}
							></i>
						</span>
						<h1>Two-factor sign-in</h1>
						<p class="muted">
							{useRecovery
								? 'Enter one of your recovery codes. Each works once.'
								: 'Enter the six-digit code from your authenticator app.'}
						</p>
					</header>

					{#if error}
						<Alert>{error}</Alert>
					{/if}

					{#key useRecovery}
						<Input
							label={useRecovery ? 'Recovery code' : 'Code'}
							bind:value={code}
							inputmode={useRecovery ? 'text' : 'numeric'}
							autocomplete="one-time-code"
							autocapitalize="none"
							spellcheck={false}
							maxlength={useRecovery ? 16 : 7}
							disabled={submitting}
						/>
					{/key}

					<Button type="submit" size="lg" loading={submitting} disabled={!canVerify}>
						{submitting ? 'Verifying…' : 'Verify'}
					</Button>

					<div class="links">
						<button
							type="button"
							class="link"
							onclick={() => {
								useRecovery = !useRecovery;
								code = '';
							}}
						>
							{useRecovery ? 'Use the authenticator app' : 'Use a recovery code'}
						</button>
						<button type="button" class="link" onclick={startOver}>Start over</button>
					</div>
				</form>
			{:else}
				<form onsubmit={signIn}>
					<header>
						<span class="mark">
							<i
								class="ri-shield-keyhole-line"
								style:font-size="20px"
								role="img"
								aria-label={BRAND.name}
							></i>
						</span>
						<h1>{BRAND.name}</h1>
						<p class="muted">Sign in to the admin panel</p>
					</header>

					{#if error}
						<Alert>{error}</Alert>
					{/if}

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
			{/if}
		</Card>
	</div>
</main>

<style>
	main {
		display: grid;
		place-items: center;
		min-height: 100dvh;
		padding: var(--space-5);
	}

	.corner {
		position: fixed;
		top: var(--space-3);
		right: var(--space-3);
		color: var(--color-text-hint);
	}

	.panel {
		width: 100%;
		max-width: 22rem;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		padding: var(--space-2);
	}

	header {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.mark {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		margin-bottom: var(--space-2);
		border-radius: var(--radius-md);
		background: var(--color-accent);
		color: var(--color-accent-text);
	}

	header h1 {
		font-size: var(--text-lg);
	}

	.muted {
		color: var(--color-text-hint);
		font-size: var(--text-base);
	}

	.links {
		display: flex;
		justify-content: space-between;
		gap: var(--space-2);
	}

	.link {
		padding: 0;
		border: 0;
		background: none;
		color: var(--color-text-hint);
		font: inherit;
		font-size: var(--text-sm);
		cursor: pointer;
	}

	.link:hover {
		color: var(--color-text);
		text-decoration: underline;
	}
</style>
