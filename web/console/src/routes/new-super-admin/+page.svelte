<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApi, ApiError, setupApi } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		Alert,
		Button,
		Card,
		Input,
		PasswordInput,
		ThemeToggle
	} from '#lib/components/ui/index.js';

	let firstName = $state('');
	let lastName = $state('');
	let email = $state('');
	let password = $state('');
	let error = $state('');
	let creating = $state(false);

	const canSubmit = $derived(
		firstName.trim() !== '' && email.trim() !== '' && password.length >= 10 && !creating
	);

	async function create(event: SubmitEvent) {
		event.preventDefault();
		if (!canSubmit) return;

		error = '';
		creating = true;

		try {
			await setupApi.create({
				email: email.trim(),
				password,
				first_name: firstName.trim(),
				last_name: lastName.trim()
			});

			const result = await adminApi.login(email.trim(), password);

			const target = result.next === 'enroll' ? '/mfa-setup' : '/(panel)';
			await goto(resolve(target), { replace: true, refreshAll: true });
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Could not create the account';
			creating = false;
		}
	}
</script>

<svelte:head><title>Set up {BRAND.name}</title></svelte:head>

<main>
	<div class="corner">
		<ThemeToggle />
	</div>

	<div class="panel">
		<Card padded>
			<form onsubmit={create}>
				<header>
					<span class="mark">
						<i
							class="ri-shield-keyhole-line"
							style:font-size="20px"
							role="img"
							aria-label={BRAND.name}
						></i>
					</span>
					<h1>Set up {BRAND.name}</h1>
					<p class="muted">
						This panel has no administrator yet. The account you make here is the super admin: it
						can do everything, including making the others.
					</p>
				</header>

				{#if error}
					<Alert>{error}</Alert>
				{/if}

				<div class="names">
					<Input label="First name" bind:value={firstName} disabled={creating} required />
					<Input label="Last name" bind:value={lastName} disabled={creating} />
				</div>

				<Input
					label="Email"
					bind:value={email}
					type="email"
					autocomplete="email"
					autocapitalize="none"
					spellcheck={false}
					hint="This is what you will sign in with."
					disabled={creating}
					required
				/>

				<PasswordInput
					label="Password"
					bind:value={password}
					autocomplete="new-password"
					disabled={creating}
					required
				/>
				<p class="muted small">At least 10 characters. Nothing else is asked of it.</p>

				<Button type="submit" size="lg" loading={creating} disabled={!canSubmit}>
					{creating ? 'Creating…' : 'Create the super admin'}
				</Button>
			</form>
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
	}

	.panel {
		width: 100%;
		max-width: 28rem;
	}

	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	header {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin-bottom: var(--space-2);
	}

	.mark {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		border-radius: var(--radius-md);
		background: var(--color-accent);
		color: var(--color-accent-text);
	}

	h1 {
		margin: 0;
		font-size: var(--text-xl);
	}

	.muted {
		color: var(--color-text-hint);
		font-size: var(--text-base);
		line-height: 1.5;
	}

	.small {
		margin-top: calc(var(--space-2) * -1);
		font-size: var(--text-sm);
	}

	.names {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-2);
	}

	@media (max-width: 30rem) {
		.names {
			grid-template-columns: 1fr;
		}
	}
</style>
