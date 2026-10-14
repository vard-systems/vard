<script lang="ts">
	import { MIN_ADMIN_PASSWORD } from '#lib/constants.js';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { RiAddLine, RiCloseLine, RiMailLine, RiShieldKeyholeLine } from 'svelte-remixicon';
	import {
		adminsApi,
		type AdminInput,
		type AdminPermission,
		type AdminRecord,
		type AdminRole,
		type Application
	} from '#lib/api/index.js';
	import {
		Badge,
		Button,
		ConfirmDialog,
		FullscreenDialog,
		FormSection,
		Icon,
		IconButton,
		Input,
		PasswordInput,
		Select,
		type SelectOption,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { grantedBy } from './permissions';

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

	const PANEL = 'panel';

	type Row = { key: number; role: string; scope: string };

	let email = $state('');
	let firstName = $state('');
	let lastName = $state('');
	let status = $state<AdminInput['status']>('active');
	let password = $state('');
	let confirmPassword = $state('');
	let rows = $state<Row[]>([]);
	let nextKey = 0;

	let saving = $state(false);
	let confirmingReset = $state(false);
	let resetting = $state(false);

	async function resetMfa() {
		if (!admin) return;
		resetting = true;

		try {
			await adminsApi.resetMfa(admin.id);
			notify.success(
				'Two-factor sign-in reset',
				'They set up a new authenticator at their next sign-in.'
			);
			await queryClient.invalidateQueries({ queryKey: keys.admins.all });
			open = false;
		} catch (err) {
			notify.error(err, 'Could not reset two-factor sign-in');
		} finally {
			resetting = false;
			confirmingReset = false;
		}
	}

	const editing = $derived(admin !== null);

	/** Editing yourself: the server refuses a change that would lock you out,
	    so the panel does not offer one. */
	const isSelf = $derived(admin?.id === self);

	const statuses: SelectOption<AdminInput['status']>[] = [
		{ value: 'active', label: 'Active', description: 'May sign in' },
		{ value: 'suspended', label: 'Suspended', description: 'Blocked for now; can be reactivated' },
		{ value: 'disabled', label: 'Disabled', description: 'Blocked for good' }
	];

	const superRole = $derived(roles.find((role) => role.is_builtin));

	const roleOptions = $derived<SelectOption[]>(
		roles.map((role) => ({
			value: role.id,
			label: role.name,
			description: role.is_builtin ? 'everything' : `${role.permissions.length} permissions`
		}))
	);

	const scopeOptions = $derived<SelectOption[]>([
		{ value: PANEL, label: 'Whole panel', description: 'Every application, and everything else' },
		...applications.map((app) => ({ value: app.id, label: app.name, description: app.client_id }))
	]);

	const mismatch = $derived(confirmPassword !== '' && password !== confirmPassword);
	const tooShort = $derived(password !== '' && password.length < MIN_ADMIN_PASSWORD);

	const canSubmit = $derived(
		!saving &&
			email.trim() !== '' &&
			firstName.trim() !== '' &&
			password === confirmPassword &&
			!tooShort &&
			rows.every((row) => row.role !== '') &&
			(editing || password !== '')
	);

	const panelPermissions = $derived(
		grantedBy(
			rows.filter((row) => row.scope === PANEL).map((row) => row.role),
			roles,
			catalog
		)
	);

	const scopedPermissions = $derived(
		applications
			.map((app) => ({
				app,
				permissions: grantedBy(
					rows.filter((row) => row.scope === app.id).map((row) => row.role),
					roles,
					catalog,
					true
				).filter((name) => !panelPermissions.includes(name))
			}))
			.filter((it) => rows.some((row) => row.scope === it.app.id))
	);

	/** Whether a scoped role grants something that only counts for the whole
	    panel, which is worth pointing out: it will not apply. */
	function unscoped(row: Row): string[] {
		if (row.scope === PANEL) return [];

		const role = roles.find((it) => it.id === row.role);
		const scopable = new Set(catalog.filter((it) => it.scopable).map((it) => it.name));

		return (role?.permissions ?? []).filter((name) => !scopable.has(name));
	}

	function locked(row: Row): boolean {
		return isSelf && row.role === superRole?.id && row.scope === PANEL;
	}

	$effect(() => {
		if (!open) return;

		email = admin?.email ?? '';
		firstName = admin?.first_name ?? '';
		lastName = admin?.last_name ?? '';
		status = admin && admin.status !== 'invited' ? admin.status : 'active';
		password = '';
		confirmPassword = '';
		rows = (admin?.assignments ?? []).map((a) => ({
			key: nextKey++,
			role: a.role.id,
			scope: a.application?.id ?? PANEL
		}));
	});

	function addRow() {
		rows = [...rows, { key: nextKey++, role: '', scope: PANEL }];
	}

	function removeRow(key: number) {
		rows = rows.filter((row) => row.key !== key);
	}

	function setRole(key: number, role: string) {
		rows = rows.map((row) =>
			row.key === key ? { ...row, role, scope: role === superRole?.id ? PANEL : row.scope } : row
		);
	}

	function setScope(key: number, scope: string) {
		rows = rows.map((row) => (row.key === key ? { ...row, scope } : row));
	}

	function payload(): AdminInput {
		return {
			email: email.trim(),
			first_name: firstName.trim(),
			last_name: lastName.trim(),
			status,
			assignments: rows.map((row) => ({
				role_id: row.role,
				application_id: row.scope === PANEL ? null : row.scope
			})),
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
			icon={RiMailLine}
			bind:value={email}
			type="email"
			autocomplete="off"
			placeholder="moderator@example.com"
			required
		/>

		<div class="names">
			<Input label="First name" bind:value={firstName} required />
			<Input label="Last name" bind:value={lastName} />
		</div>

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
		<div class="names">
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
		</div>

		{#if mismatch}
			<p class="hint danger">The passwords do not match.</p>
		{:else if tooShort}
			<p class="hint danger">At least {MIN_ADMIN_PASSWORD} characters.</p>
		{/if}
	</FormSection>

	<FormSection
		title="Roles"
		description="Each role is held for the whole panel or for one application. A role held for one application grants only its application permissions, and only there."
	>
		<div class="rows">
			{#each rows as row (row.key)}
				{@const ignored = unscoped(row)}
				<div class="row">
					<div class="role">
						<Select
							label="Role"
							value={row.role}
							options={roleOptions}
							placeholder="Choose a role"
							readOnly={locked(row)}
							onChange={(value) => setRole(row.key, value)}
						/>
					</div>

					<div class="scope">
						<Select
							label="For"
							value={row.scope}
							options={scopeOptions}
							readOnly={locked(row) || row.role === superRole?.id}
							onChange={(value) => setScope(row.key, value)}
						/>
					</div>

					<IconButton
						icon={RiCloseLine}
						label="Remove this role"
						onclick={() => removeRow(row.key)}
						disabled={locked(row)}
					/>

					{#if locked(row)}
						<p class="note">You cannot take away your own super_admin role.</p>
					{:else if ignored.length > 0}
						<p class="note">
							Only applies for the whole panel, so not granted here: {ignored.join(', ')}
						</p>
					{/if}
				</div>
			{:else}
				<p class="hint">No roles: they could sign in but see nothing.</p>
			{/each}
		</div>

		<div>
			<Button size="sm" variant="subtle" onclick={addRow}>
				<Icon icon={RiAddLine} />
				Add a role
			</Button>
		</div>

		<div class="summary">
			<span class="label">Whole panel</span>
			<div class="badges">
				{#each panelPermissions as permission (permission)}
					<Badge>{permission}</Badge>
				{:else}
					<span class="hint">nothing</span>
				{/each}
			</div>

			{#each scopedPermissions as { app, permissions } (app.id)}
				<span class="label">{app.name}</span>
				<div class="badges">
					{#each permissions as permission (permission)}
						<Badge>{permission}</Badge>
					{:else}
						<span class="hint">nothing beyond the whole panel</span>
					{/each}
				</div>
			{/each}
		</div>
	</FormSection>

	{#if admin && !isSelf}
		<FormSection
			title="Two-factor sign-in"
			description="Whether they sign in with a code from an authenticator app."
		>
			<div class="mfa">
				<Badge>{admin.mfa_enabled ? 'On' : 'Off'}</Badge>
				{#if admin.mfa_enabled}
					<Button size="sm" variant="subtle" onclick={() => (confirmingReset = true)}>
						Reset two-factor
					</Button>
				{/if}
			</div>
			<p class="hint">
				For someone who lost their phone and their recovery codes. They are signed out, and set it
				up again at their next sign-in where it is required.
			</p>
		</FormSection>
	{/if}

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>

		<Button type="submit" loading={saving} disabled={!canSubmit}>
			{saving ? 'Saving…' : editing ? 'Save changes' : 'Create administrator'}
		</Button>
	{/snippet}
</FullscreenDialog>

<ConfirmDialog
	bind:open={confirmingReset}
	title="Reset two-factor sign-in?"
	description={`${admin?.email ?? 'This administrator'} loses their authenticator and recovery codes, and is signed out everywhere. They set it up again at their next sign-in where it is required.`}
	tone="warning"
	icon={RiShieldKeyholeLine}
	confirmLabel="Reset two-factor"
	busy={resetting}
	onConfirm={resetMfa}
/>

<style>
	.hint {
		margin: calc(var(--space-2) * -1) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.4;
	}

	.hint.danger {
		color: var(--color-danger);
	}

	.mfa {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
	}

	.names {
		display: grid;
		grid-template-columns: 1fr 1fr;
		align-items: start;
		gap: var(--space-3);
	}

	.rows {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.row {
		display: grid;
		grid-template-columns: 1fr 1fr auto;
		align-items: center;
		gap: var(--space-2);
	}

	.note {
		grid-column: 1 / -1;
		margin: calc(var(--space-2) * -1) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-style: italic;
	}

	.summary {
		display: grid;
		grid-template-columns: 8rem 1fr;
		align-items: start;
		gap: var(--space-2) var(--space-3);
	}

	.label {
		padding-top: 4px;
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.badges {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}

	@media (max-width: 30rem) {
		.names,
		.row,
		.summary {
			grid-template-columns: 1fr;
		}
	}
</style>
