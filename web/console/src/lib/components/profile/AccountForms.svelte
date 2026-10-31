<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidate } from '$app/navigation';
	import type { Admin } from '#lib/api/index.js';
	import {
		Button,
		FieldGrid,
		Input,
		Panel,
		PasswordInput,
		Thumb,
		notify
	} from '#lib/components/ui/index.js';
	import { ADMIN_DEPENDENCY, MIN_ADMIN_PASSWORD } from '#lib/constants.js';
	import { longEnoughForAdmin } from '#lib/utils/password.js';
	import { initials } from '#lib/utils/format.js';
	import { submit } from '#lib/forms.js';

	type Props = { admin: Admin };

	let { admin }: Props = $props();

	// svelte-ignore state_referenced_locally
	let firstName = $state(admin.first_name);
	// svelte-ignore state_referenced_locally
	let lastName = $state(admin.last_name);
	// svelte-ignore state_referenced_locally
	let email = $state(admin.email);
	// svelte-ignore state_referenced_locally
	let avatarUrl = $state(admin.avatar_url);
	/** Asked for only when the address changes: it is what they sign in
	    with, so a session left open is not enough to move it. */
	let emailPassword = $state('');
	let savingProfile = $state(false);

	const emailChanged = $derived(email.trim().toLowerCase() !== admin.email.toLowerCase());
	const profileChanged = $derived(
		firstName.trim() !== admin.first_name ||
			lastName.trim() !== admin.last_name ||
			avatarUrl.trim() !== admin.avatar_url ||
			emailChanged
	);

	const preview = $derived(/^https?:\/\/\S+$/.test(avatarUrl.trim()) ? avatarUrl.trim() : '');
	const monogram = $derived(initials(`${firstName} ${lastName}`.trim() || email));

	const saveProfile = submit<unknown>({
		input: () => [
			{
				first_name: firstName.trim(),
				last_name: lastName.trim(),
				email: email.trim(),
				avatar_url: avatarUrl.trim(),
				current_password: emailChanged ? emailPassword : undefined
			}
		],
		when: () => !savingProfile,
		pending: (busy) => (savingProfile = busy),
		done: async () => {
			emailPassword = '';
			notify.success('Your profile is saved');
			await invalidate(ADMIN_DEPENDENCY);
		},
		failed: (err) => notify.error(err, 'Could not save your profile')
	});

	let currentPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let changingPassword = $state(false);

	/** Says what is wrong with the new password before it is sent; "" when nothing is. */
	function passwordProblem(): string {
		if (!longEnoughForAdmin(newPassword)) {
			return `The new password must be at least ${MIN_ADMIN_PASSWORD} characters.`;
		}
		if (newPassword !== confirmPassword) return 'The two new passwords are not the same.';
		return '';
	}

	const changePassword = submit<unknown>({
		input: () => [currentPassword, newPassword],
		when: () => {
			const problem = passwordProblem();
			if (problem) notify.problem(problem);
			return problem === '' && !changingPassword;
		},
		pending: (busy) => (changingPassword = busy),
		done: () => {
			currentPassword = newPassword = confirmPassword = '';
			notify.success('Your password is changed', 'Your other sessions are signed out.');
		},
		failed: (err) => notify.error(err, 'Could not change your password')
	});
</script>

<div class="sections">
	<Panel title="Profile" icon="ri-user-line">
		<form class="form" method="POST" action="/actions/me?/updateProfile" use:enhance={saveProfile}>
			<FieldGrid>
				<Input label="First name" bind:value={firstName} autocomplete="given-name" required />
				<Input label="Last name" bind:value={lastName} autocomplete="family-name" />
			</FieldGrid>
			<Input
				label="Email"
				type="email"
				bind:value={email}
				autocomplete="email"
				hint="You sign in with it, and anything about this account is sent to it."
				required
			/>
			<div class="avatar">
				<span class="picture">
					<Thumb src={preview} text={monogram} size="md" shape="circle" tone="accent" />
				</span>
				<Input
					label="Avatar URL"
					icon="ri-image-line"
					bind:value={avatarUrl}
					type="url"
					maxlength={512}
					placeholder="https://example.com/me.png"
					hint="A picture of you for the header and menus. Empty, your initials stand in."
				/>
			</div>
			{#if emailChanged}
				<PasswordInput
					label="Current password, to change your email"
					bind:value={emailPassword}
					required
				/>
			{/if}

			<div class="actions">
				<Button
					type="submit"
					size="sm"
					loading={savingProfile}
					disabled={savingProfile ||
						!profileChanged ||
						firstName.trim() === '' ||
						(emailChanged && emailPassword === '')}
				>
					Save profile
				</Button>
			</div>
		</form>
	</Panel>

	<Panel title="Password" icon="ri-lock-password-line">
		<form
			class="form"
			method="POST"
			action="/actions/me?/changePassword"
			use:enhance={changePassword}
		>
			<PasswordInput label="Current password" bind:value={currentPassword} required />
			<FieldGrid>
				<PasswordInput
					label="New password"
					bind:value={newPassword}
					autocomplete="new-password"
					required
				/>
				<PasswordInput
					label="Repeat the new password"
					bind:value={confirmPassword}
					autocomplete="new-password"
					required
				/>
			</FieldGrid>
			<p class="quiet">
				{`At least ${MIN_ADMIN_PASSWORD} characters. Changing it signs you out everywhere but here.`}
			</p>

			<div class="actions">
				<Button
					type="submit"
					size="sm"
					loading={changingPassword}
					disabled={changingPassword ||
						currentPassword === '' ||
						newPassword === '' ||
						confirmPassword === ''}
				>
					Change password
				</Button>
			</div>
		</form>
	</Panel>
</div>

<style>
	.sections {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.quiet {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.avatar {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
	}

	/* As tall as the field beside it, so the picture sits on its middle
	   whether or not a hint hangs under the field. */
	.picture {
		display: flex;
		flex: none;
		align-items: center;
		height: var(--field-height);
	}

	.avatar > :global(:last-child) {
		flex: 1;
		min-width: 0;
	}

	.actions {
		display: flex;
		justify-content: flex-end;
	}
</style>
