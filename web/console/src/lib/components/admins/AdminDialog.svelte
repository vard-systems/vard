<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import {
		adminsApi,
		type AdminInput,
		type AdminPermission,
		type AdminRecord,
		type AdminRole,
		type Application
	} from '#lib/api/index.js';
	import {
		Button,
		FieldGrid,
		FullscreenDialog,
		FormSection,
		Input,
		PasswordInput,
		Select,
		type SelectOption,
		notify
	} from '#lib/components/ui/index.js';
	import { MIN_ADMIN_PASSWORD } from '#lib/constants.js';
	import { keys } from '#lib/query/index.js';
	import { longEnoughForAdmin } from '#lib/utils/password.js';
	import AdminMfaReset from './AdminMfaReset.svelte';
	import AdminRoles from './AdminRoles.svelte';
	import { assignmentsOf, rowsFrom, type Row } from './assignments';

	type Props = {
		admin: AdminRecord | null;
		self: string;
		roles: AdminRole[];
		applications: Application[];
		catalog: AdminPermission[];
		open: boolean;
	};

	let { admin, self, roles, applications, catalog, open = $bindable(false) }: Props = $props();

	const queryClient = useQueryClient();

	const statuses: SelectOption<AdminInput['status']>[] = [
		{ value: 'active', label: 'Active', description: 'May sign in' },
		{ value: 'suspended', label: 'Suspended', description: 'Blocked for now; can be reactivated' },
		{ value: 'disabled', label: 'Disabled', description: 'Blocked for good' }
	];

	let email = $state('');
	let firstName = $state('');
	let lastName = $state('');
	let status = $state<AdminInput['status']>('active');
	let password = $state('');
	let confirmPassword = $state('');
	let rows = $state<Row[]>([]);
	let saving = $state(false);

	let lastKey = 0;
	const nextKey = () => lastKey++;

	const editing = $derived(admin !== null);

	/** Editing yourself: the server refuses a change that would lock you out,
	    so the panel does not offer one. */
	const isSelf = $derived(admin?.id === self);

	const mismatch = $derived(confirmPassword !== '' && password !== confirmPassword);
	const tooShort = $derived(password !== '' && !longEnoughForAdmin(password));

	const canSubmit = $derived(
		!saving &&
			email.trim() !== '' &&
			firstName.trim() !== '' &&
			password === confirmPassword &&
			!tooShort &&
			rows.every((row) => row.role !== '') &&
			(editing || password !== '')
	);

	$effect(() => {
		if (!open) return;

		email = admin?.email ?? '';
		firstName = admin?.first_name ?? '';
		lastName = admin?.last_name ?? '';
		status = admin && admin.status !== 'invited' ? admin.status : 'active';
		password = '';
		confirmPassword = '';
		rows = rowsFrom(admin?.assignments ?? [], nextKey);
	});

	function payload(): AdminInput {
		return {
			email: email.trim(),
			first_name: firstName.trim(),
			last_name: lastName.trim(),
			status,
			assignments: assignmentsOf(rows),
			...(password !== '' ? { password, confirm_password: confirmPassword } : {})
		};
	}

	const save = createMutation(() => ({
		mutationFn: () => (admin ? adminsApi.update(admin.id, payload()) : adminsApi.create(payload())),
		onSuccess: async () => {
			notify.success(admin ? 'Administrator saved' : 'Administrator created');
			await queryClient.invalidateQueries({ queryKey: keys.admins.all });
			open = false;
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this administrator');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (!canSubmit) return;

		saving = true;
		save.mutate();
	}
</script>

<FullscreenDialog
	bind:open
	title={admin ? admin.full_name.trim() || admin.email : 'New administrator'}
	meta={admin ? (isSelf ? `${admin.email} · you` : admin.email) : undefined}
	onsubmit={submit}
>
	<FormSection
		title="Account"
		description="Who the administrator is, and the address they sign in with."
	>
		<Input
			label="Email"
			icon="ri-mail-line"
			bind:value={email}
			type="email"
			autocomplete="off"
			placeholder="moderator@example.com"
			required
		/>

		<FieldGrid>
			<Input label="First name" bind:value={firstName} required />
			<Input label="Last name" bind:value={lastName} />
		</FieldGrid>

		{#if editing}
			<Input label="Administrator ID" value={admin?.id ?? ''} readOnly copyable />
		{/if}

		<Select
			label="Status"
			bind:value={status}
			options={statuses}
			readOnly={isSelf}
			hint={isSelf
				? 'You cannot suspend your own account.'
				: 'Suspending or disabling an account signs it out everywhere.'}
		/>
	</FormSection>

	<FormSection
		title="Password"
		description={editing
			? `Leave both fields empty to keep the current password.${isSelf ? '' : ' A new one signs them out everywhere.'}`
			: `At least ${MIN_ADMIN_PASSWORD} characters.`}
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
			<p class="problem">The passwords do not match.</p>
		{:else if tooShort}
			<p class="problem">At least {MIN_ADMIN_PASSWORD} characters.</p>
		{/if}
	</FormSection>

	<AdminRoles bind:rows {roles} {applications} {catalog} {isSelf} {nextKey} />

	{#if admin && !isSelf}
		<AdminMfaReset {admin} onreset={() => (open = false)} />
	{/if}

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>

		<Button type="submit" loading={saving} disabled={!canSubmit}>
			{saving ? 'Saving…' : editing ? 'Save changes' : 'Create administrator'}
		</Button>
	{/snippet}
</FullscreenDialog>

<style>
	.problem {
		margin: calc(var(--space-2) * -1) 0 0;
		color: var(--color-danger);
		font-size: var(--text-sm);
	}
</style>
