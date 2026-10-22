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
		FullscreenDialog,
		FieldGrid,
		FormSection,
		Input,
		LinkButton,
		List,
		ListItem,
		PasswordInput,
		SwitchField,
		Tabs,
		Tag,
		Thumb,
		notify
	} from '#lib/components/ui/index.js';
	import { keys, roleMappingsOptions } from '#lib/query/index.js';
	import { markFor as providerMark } from '#lib/components/social/providers.js';
	import { formatDate, formatRelative } from '#lib/utils/format.js';
	import RoleMappings from '#lib/components/roles/RoleMappings.svelte';
	import { additional } from './fields';
	import FieldInput from './FieldInput.svelte';

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

	let current = $state<UserRecord | null>(null);

	let tab = $state('user');

	let created = $state(false);

	let email = $state('');
	let isEmailVerified = $state(false);
	let firstName = $state('');
	let lastName = $state('');
	let isActive = $state(true);

	let password = $state('');
	let confirmPassword = $state('');
	let isPasswordTemporary = $state(false);

	let values = $state<Record<string, string | boolean>>({});

	let saving = $state(false);

	const editing = $derived(current !== null);

	let disconnecting = $state('');

	async function disconnect(identity: string) {
		if (!current) return;

		disconnecting = identity;

		try {
			await usersApi.disconnect(current.id, identity);
			notify.success('Provider disconnected');
			await queryClient.invalidateQueries({ queryKey: keys.users.all });

			current = {
				...current,
				social_accounts: current.social_accounts.filter((account) => account.id !== identity)
			};
		} catch (err) {
			notify.error(err, 'Could not disconnect this provider');
		} finally {
			disconnecting = '';
		}
	}

	const mappings = createQuery(() => ({
		...roleMappingsOptions(current?.id ?? ''),
		enabled: open && current !== null
	}));

	const mismatch = $derived(confirmPassword !== '' && password !== confirmPassword);

	const hasPassword = $derived(!editing || password !== '' || (current?.has_password ?? false));

	const canSubmit = $derived(
		!saving &&
			editable &&
			email.trim() !== '' &&
			password === confirmPassword &&
			(editing || password !== '')
	);

	const extras = $derived(additional(fields));
	const details = $derived(extras.filter((field) => field.type !== 'bool'));
	const flags = $derived(extras.filter((field) => field.type === 'bool'));

	$effect(() => {
		if (!open) {
			tab = 'user';
			return;
		}

		current = user;
		tab = 'user';
		created = false;
		untrack(() => fill(user));
	});

	function fill(record: UserRecord | null) {
		email = record?.email ?? '';
		isEmailVerified = record?.is_email_verified ?? false;
		firstName = record?.first_name ?? '';
		lastName = record?.last_name ?? '';
		isActive = record?.is_active ?? true;
		password = '';
		confirmPassword = '';
		isPasswordTemporary = record ? record.is_password_temporary : true;

		values = Object.fromEntries(
			extras.map((field) => {
				const stored = record?.data?.[field.name];

				if (field.type === 'bool') return [field.name, stored === true];
				const text = typeof stored === 'string' || typeof stored === 'number' ? String(stored) : '';
				if (field.type === 'date') return [field.name, text.slice(0, 10)];

				return [field.name, text];
			})
		);
	}

	function payload() {
		const data: Record<string, unknown> = {};

		for (const field of extras) {
			const value = values[field.name];

			if (field.type === 'bool') {
				data[field.name] = value === true;
			} else if (String(value ?? '').trim() !== '') {
				data[field.name] = value;
			}
		}

		return {
			email: email.trim(),
			is_email_verified: isEmailVerified,
			first_name: firstName.trim(),
			last_name: lastName.trim(),
			is_active: isActive,
			is_password_temporary: isPasswordTemporary && hasPassword,
			...(password !== '' ? { password, confirm_password: confirmPassword } : {}),
			data
		};
	}

	const save = createMutation(() => ({
		mutationFn: () =>
			current ? usersApi.update(current.id, payload()) : usersApi.create(payload()),
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

			current = result.user;
			fill(result.user);
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
							bind:value={email}
							type="email"
							autocomplete="off"
							placeholder="user@example.com"
							required
						/>

						<FieldGrid>
							<Input label="First name" bind:value={firstName} />
							<Input label="Last name" bind:value={lastName} />
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
							: 'At least 8 characters.'}
					>
						<FieldGrid>
							<PasswordInput
								label={editing ? 'New password' : 'Password'}
								bind:value={password}
								autocomplete="new-password"
								required={!editing}
							/>
							<PasswordInput
								label="Confirm password"
								bind:value={confirmPassword}
								autocomplete="new-password"
								required={!editing || password !== ''}
							/>
						</FieldGrid>

						{#if mismatch}
							<p class="mismatch">The passwords do not match.</p>
						{/if}

						<SwitchField
							label="Is Temporary"
							description="The user has to choose a new password the next time they sign in."
							bind:checked={isPasswordTemporary}
							disabled={!hasPassword}
						/>
					</FormSection>

					{#if editing && (current?.social_accounts ?? []).length > 0}
						<FormSection
							title="Signs in with"
							description="Accounts elsewhere that reach this one. Disconnecting one leaves the account itself alone."
						>
							<List bordered label="Connected providers">
								{#each current?.social_accounts ?? [] as account (account.id)}
									<ListItem
										title={account.provider}
										description={account.email || `Connected ${formatDate(account.connected_at)}`}
									>
										{#snippet lead()}
											<Thumb icon={providerMark(account.kind)} />
										{/snippet}

										{#snippet end()}
											{#if account.last_login_at}
												<Tag small>used {formatRelative(account.last_login_at)}</Tag>
											{/if}

											{#if editable}
												<Button
													size="sm"
													variant="subtle"
													colorPalette="danger"
													loading={disconnecting === account.id}
													disabled={disconnecting !== ''}
													onclick={() => disconnect(account.id)}
												>
													Disconnect
												</Button>
											{/if}
										{/snippet}
									</ListItem>
								{/each}
							</List>
						</FormSection>
					{/if}

					<FormSection
						title="Status"
						description="Whether they can sign in, and whether their address is confirmed."
					>
						<div class="switches">
							<SwitchField
								label="Is Active"
								description="An inactive user cannot sign in to any application."
								bind:checked={isActive}
							/>
							<SwitchField
								label="Is Email Verified"
								description="Whether the user has confirmed they own this address."
								bind:checked={isEmailVerified}
							/>
						</div>
					</FormSection>

					{#if extras.length > 0}
						<FormSection
							title="Additional fields"
							description="The fields this organisation keeps about its users."
						>
							{#each details as field (field.id)}
								<FieldInput {field} bind:value={values[field.name]} />
							{/each}

							{#if flags.length > 0}
								<div class="flags">
									{#each flags as field (field.id)}
										<FieldInput {field} bind:value={values[field.name]} />
									{/each}
								</div>
							{/if}
						</FormSection>
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

	.flags {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: var(--space-2) var(--space-4);
	}
</style>
