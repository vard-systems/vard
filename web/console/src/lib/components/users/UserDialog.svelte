<script lang="ts">
	import { untrack } from 'svelte';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { resolve } from '$app/paths';
	import {
		usersApi,
		type Admin,
		type Application,
		type Role,
		type UserField,
		type UserRecord
	} from '#lib/api/index.js';
	import {
		Alert,
		Button,
		FieldGrid,
		FormSection,
		FullscreenDialog,
		Input,
		LinkButton,
		PasswordInput,
		SwitchField,
		Tabs,
		notify
	} from '#lib/components/ui/index.js';
	import { MIN_USER_PASSWORD } from '#lib/constants.js';
	import { keys, roleMappingsOptions } from '#lib/query/index.js';
	import RoleMappings from '#lib/components/roles/RoleMappings.svelte';
	import { additional } from './fields';
	import { UserForm } from './form.svelte';
	import UserExtraFields from './UserExtraFields.svelte';
	import UserProviders from './UserProviders.svelte';
	import UserTwoStep from './UserTwoStep.svelte';

	type Props = {
		user: UserRecord | null;
		fields: UserField[];
		applications: Application[];
		roles: Role[];
		admin: Admin;
		open: boolean;
		editable?: boolean;
	};

	let {
		user,
		fields,
		applications,
		roles,
		admin,
		open = $bindable(false),
		editable = true
	}: Props = $props();

	const queryClient = useQueryClient();

	const form = new UserForm();

	/** The user being edited: the one opened, or the one just created. */
	let current = $state<UserRecord | null>(null);
	let tab = $state('user');
	let created = $state(false);
	let saving = $state(false);

	const editing = $derived(current !== null);
	const extras = $derived(additional(fields));

	/** Whether the user will have a password once saved: only a password can be temporary. */
	const hasPassword = $derived(
		!editing || form.password !== '' || (current?.has_password ?? false)
	);

	const canSubmit = $derived(
		!saving &&
			editable &&
			form.email.trim() !== '' &&
			form.password === form.confirmPassword &&
			(editing || form.password !== '')
	);

	const mappings = createQuery(() => ({
		...roleMappingsOptions(current?.id ?? ''),
		enabled: open && current !== null
	}));

	const tabs = $derived([
		{ value: 'user', label: 'User', icon: 'ri-user-line' },
		{
			value: 'roles',
			label: 'Role mapping',
			icon: 'ri-shield-user-line',
			count: mappings.data?.filter((it) => it.assigned).length,
			disabled: !editing
		}
	]);

	$effect(() => {
		tab = 'user';
		if (!open) return;

		current = user;
		created = false;
		untrack(() => form.fill(user, extras));
	});

	const save = createMutation(() => ({
		mutationFn: () => {
			const input = form.toInput(extras, hasPassword);
			return current ? usersApi.update(current.id, input) : usersApi.create(input);
		},
		onSuccess: async (result: { user: UserRecord }) => {
			notify.success(current ? 'User saved' : 'User created');
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.users.all }),
				queryClient.invalidateQueries({ queryKey: keys.roles.all })
			]);

			if (current) {
				open = false;
				return;
			}

			// Created: stay open on its roles, which is what comes next.
			current = result.user;
			form.fill(result.user, extras);
			created = true;
			tab = 'roles';
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this user');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (tab !== 'user' || !canSubmit) return;

		saving = true;
		save.mutate();
	}
</script>

<FullscreenDialog
	bind:open
	title={current
		? `${current.first_name} ${current.last_name}`.trim() || current.email
		: 'New user'}
	meta={current?.email}
	onsubmit={submit}
>
	{#if !editing || created}
		<div class="intro">
			{#if created}
				<Alert tone="success">
					User created, with the default roles. Assign any others here, or close this window.
				</Alert>
			{:else}
				<Alert tone="info">
					Create the user first; roles are given on the Role mapping tab afterwards.
				</Alert>
			{/if}
		</div>
	{/if}

	<Tabs {tabs} bind:value={tab} label="User sections">
		{#snippet panel(value)}
			{#if value === 'user'}
				<fieldset disabled={!editable}>
					<FormSection
						title="Account"
						description="Who the user is, and the address they sign in with."
					>
						<Input
							label="Email"
							icon="ri-mail-line"
							bind:value={form.email}
							type="email"
							autocomplete="off"
							placeholder="user@example.com"
							required
						/>

						<FieldGrid>
							<Input label="First name" bind:value={form.firstName} />
							<Input label="Last name" bind:value={form.lastName} />
						</FieldGrid>

						{#if editing}
							<Input label="User ID" value={current?.id ?? ''} readOnly copyable />
						{/if}
					</FormSection>

					<FormSection
						title="Password"
						description={editing
							? current?.has_password
								? 'Leave both fields empty to keep the current password.'
								: 'This user has no password yet.'
							: `At least ${MIN_USER_PASSWORD} characters.`}
					>
						<FieldGrid>
							<PasswordInput
								label={editing ? 'New password' : 'Password'}
								bind:value={form.password}
								autocomplete="new-password"
								required={!editing}
							/>
							<PasswordInput
								label="Confirm password"
								bind:value={form.confirmPassword}
								autocomplete="new-password"
								required={!editing || form.password !== ''}
							/>
						</FieldGrid>

						{#if form.mismatch}
							<p class="mismatch">The passwords do not match.</p>
						{/if}

						<SwitchField
							label="Is Temporary"
							description="The user has to choose a new password the next time they sign in."
							bind:checked={form.isPasswordTemporary}
							disabled={!hasPassword}
						/>
					</FormSection>

					{#if current && (current.social_accounts ?? []).length > 0}
						<UserProviders
							user={current}
							{editable}
							ondisconnected={(updated) => (current = updated)}
						/>
					{/if}

					{#if current?.two_step_enabled}
						<UserTwoStep
							user={current}
							{editable}
							onreset={() => current && (current = { ...current, two_step_enabled: false })}
						/>
					{/if}

					<FormSection
						title="Status"
						description="Whether they can sign in, and whether their address is confirmed."
					>
						<div class="switches">
							<SwitchField
								label="Is Active"
								description="An inactive user cannot sign in to any application."
								bind:checked={form.isActive}
							/>
							<SwitchField
								label="Is Email Verified"
								description="Whether the user has confirmed they own this address."
								bind:checked={form.isEmailVerified}
							/>
						</div>
					</FormSection>

					{#if extras.length > 0}
						<UserExtraFields {extras} bind:values={form.values} />
					{/if}
				</fieldset>
			{:else if current}
				<RoleMappings userId={current.id} {roles} {applications} {admin} />
			{/if}
		{/snippet}
	</Tabs>

	{#snippet actions()}
		{#if current}
			<LinkButton
				variant="ghost"
				href={`${resolve('/(panel)/sessions')}?${new URLSearchParams({ user: current.id, email: current.email })}`}
			>
				<i class="ri-computer-line" aria-hidden="true"></i>
				Sessions
			</LinkButton>
		{/if}

		{#if editable && tab === 'user'}
			{#if !created}
				<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>
			{/if}
			<Button type="submit" loading={saving} disabled={!canSubmit}>
				{saving ? 'Saving…' : editing ? 'Save changes' : 'Create user'}
			</Button>
		{/if}
	{/snippet}
</FullscreenDialog>

<style>
	fieldset {
		min-width: 0;
		margin: 0;
		padding: 0;
		border: none;
	}

	.intro {
		margin-bottom: var(--space-4);
	}

	.mismatch {
		margin: calc(var(--space-2) * -1) 0 0;
		color: var(--color-danger);
		font-size: var(--text-sm);
	}

	.switches {
		display: flex;
		flex-direction: column;
	}
</style>
