<script lang="ts">
	import { untrack } from 'svelte';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { rolesApi, type Admin, type API, type Application, type Role } from '#lib/api/index.js';
	import {
		Button,
		FullscreenDialog,
		FormSection,
		Input,
		Select,
		SwitchField,
		Textarea,
		type SelectOption,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { submit } from '#lib/forms.js';
	import IncludedRoles from './IncludedRoles.svelte';
	import RoleScopes from './RoleScopes.svelte';
	import { canEditRoles, mayInherit, tidyName } from './roles';

	type Props = {
		role: Role | null;
		initialScope: string | null;
		roles: Role[];
		applications: Application[];
		apis: API[] | null;
		admin: Admin;
		open: boolean;
	};

	let {
		role,
		initialScope,
		roles,
		applications,
		apis,
		admin,
		open = $bindable(false)
	}: Props = $props();

	const queryClient = useQueryClient();

	let target = $state('global');
	let name = $state('');
	let description = $state('');
	let isDefault = $state(false);
	let inherits = $state<string[]>([]);
	let granted = $state<string[]>([]);

	let saving = $state(false);

	const editing = $derived(role !== null);

	const scope = $derived(
		role ? role.application_id : target === 'global' ? null : target || undefined
	);

	const writableApps = $derived(applications.filter((app) => canEditRoles(admin, app.id)));
	const mayWriteGlobal = $derived(canEditRoles(admin, null));

	const targetOptions = $derived<SelectOption[]>([
		...((role ? role.application_id === null : mayWriteGlobal)
			? [
					{
						value: 'global',
						label: 'Global',
						description: 'The same role in every application',
						icon: 'ri-global-line'
					}
				]
			: []),
		...(role ? applications.filter((app) => app.id === role.application_id) : writableApps).map(
			(app) => ({
				value: app.id,
				label: app.name,
				description: `Only in ${app.name}`,
				icon: 'ri-shield-user-line'
			})
		)
	]);

	const editable = $derived(role ? canEditRoles(admin, role.application_id) : true);

	const ready = $derived(scope !== undefined && name.trim() !== '');

	$effect(() => {
		if (!open) return;

		const start = role ? role.application_id : initialScope;
		target = start ?? 'global';

		// Start somewhere the administrator may actually make a role.
		// Read without tracking: the options are refilled when the
		// applications are, and that must not reset a form being filled in.
		const options = untrack(() => targetOptions);
		if (!role && !options.some((option) => option.value === target)) {
			target = options[0]?.value ?? '';
		}

		name = role?.name ?? '';
		description = role?.description ?? '';
		isDefault = role?.is_default ?? false;
		inherits = role?.inherits.map((inherited) => inherited.id) ?? [];
		granted = role?.api_scopes.map((scope) => scope.id) ?? [];
	});

	/** A role moved to another application keeps only what it may still include. */
	function keepAllowed() {
		const now = target === 'global' ? null : target;
		inherits = inherits.filter((id) => {
			const other = roles.find((it) => it.id === id);
			return other !== undefined && mayInherit(now, other);
		});
	}

	function payload() {
		return {
			application_id: scope ?? null,
			name: tidyName(name),
			description: description.trim(),
			is_default: isDefault,
			inherits,
			...(apis ? { api_scopes: granted } : {})
		};
	}

	const save = submit<Awaited<ReturnType<typeof rolesApi.update>>>({
		input: () => (role ? [role.id, payload()] : [payload()]),
		when: () => !(!editable || !ready),
		pending: (busy) => (saving = busy),
		done: async () => {
			notify.success(role ? 'Role saved' : 'Role created');
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.roles.all }),
				queryClient.invalidateQueries({ queryKey: keys.users.all })
			]);
			open = false;
		},
		failed: (err) => notify.error(err, 'Could not save this role')
	});
</script>

{#snippet actions()}
	<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>
	<Button type="submit" loading={saving} disabled={saving || !ready}>
		{saving ? 'Saving…' : editing ? 'Save changes' : 'Create role'}
	</Button>
{/snippet}

<FullscreenDialog
	bind:open
	actions={editable ? actions : undefined}
	title={role ? role.name : 'New role'}
	meta={role ? `${role.user_count} ${role.user_count === 1 ? 'user' : 'users'}` : undefined}
	action={role ? '/actions/roles?/update' : '/actions/roles?/create'}
	onsubmit={save}
>
	<fieldset disabled={!editable}>
		<FormSection title="Role" description="What applications check for, and where it applies.">
			<Select
				label="Applies to"
				bind:value={target}
				options={targetOptions}
				placeholder="Choose where it applies"
				readOnly={editing}
				hint={editing
					? 'A role keeps where it applies once it is made.'
					: scope === null
						? 'A global role is the same in every application.'
						: 'An application role only means something in that application.'}
				onChange={keepAllowed}
				required
			/>

			<Input
				label="Name"
				bind:value={name}
				autocomplete="off"
				autocapitalize="none"
				spellcheck={false}
				placeholder={scope === null ? 'employee' : 'editor'}
				hint="Lower case letters, numbers, dashes and underscores."
				required
			/>

			<Textarea
				label="Description"
				bind:value={description}
				rows={2}
				placeholder="Who should hold this role, and what it is for"
			/>

			{#if editing}
				<Input label="Role ID" value={role?.id ?? ''} readOnly copyable />
			{/if}

			<SwitchField
				label="Default role"
				description="Give this role to every new user automatically."
				bind:checked={isDefault}
			/>
		</FormSection>

		<IncludedRoles bind:inherits {role} {scope} {roles} {applications} {admin} {editable} />

		{#if apis}
			<RoleScopes bind:granted {apis} {editable} />
		{/if}
	</fieldset>
</FullscreenDialog>

<style>
	fieldset {
		min-width: 0;
		margin: 0;
		padding: 0;
		border: none;
	}
</style>
