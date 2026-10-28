<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApi, messageOf, setupApi } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import GateShell from '#lib/components/layout/GateShell.svelte';
	import { Alert, Button, FieldGrid, Input, PasswordInput } from '#lib/components/ui/index.js';
	import { MIN_ADMIN_PASSWORD } from '#lib/constants.js';
	import { longEnoughForAdmin } from '#lib/utils/password.js';

	let firstName = $state('');
	let lastName = $state('');
	let email = $state('');
	let password = $state('');
	let setupToken = $state('');
	let error = $state('');
	let creating = $state(false);

	const canSubmit = $derived(
		setupToken.trim() !== '' &&
			firstName.trim() !== '' &&
			email.trim() !== '' &&
			longEnoughForAdmin(password) &&
			!creating
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
				last_name: lastName.trim(),
				setup_token: setupToken.trim()
			});

			const result = await adminApi.login(email.trim(), password);

			const target = result.next === 'enroll' ? '/mfa-setup' : '/(panel)';
			await goto(resolve(target), { replace: true, refreshAll: true });
		} catch (err) {
			error = messageOf(err, 'Could not create the account');
			creating = false;
		}
	}
</script>

<svelte:head><title>Set up {BRAND.name}</title></svelte:head>

<GateShell
	title="Set up {BRAND.name}"
	width="md"
	description="This panel has no administrator yet. The account you make here is the super admin: it can do everything, including making the others."
>
	<form onsubmit={create}>
		{#if error}<Alert>{error}</Alert>{/if}

		<Input
			label="Setup token"
			bind:value={setupToken}
			autocomplete="off"
			autocapitalize="none"
			spellcheck={false}
			hint="The server wrote it to its log when it started, as setup_token. Restarting the server writes a new one."
			disabled={creating}
			required
		/>

		<FieldGrid>
			<Input label="First name" bind:value={firstName} disabled={creating} required />
			<Input label="Last name" bind:value={lastName} disabled={creating} />
		</FieldGrid>

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
			hint="At least {MIN_ADMIN_PASSWORD} characters. Nothing else is asked of it."
			disabled={creating}
			required
		/>

		<Button type="submit" size="lg" loading={creating} disabled={!canSubmit}>
			{creating ? 'Creating…' : 'Create the super admin'}
		</Button>
	</form>
</GateShell>
