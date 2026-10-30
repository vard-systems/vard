<script lang="ts">
	import { useQueryClient } from '@tanstack/svelte-query';
	import {
		adminsApi,
		type AdminPermission,
		type AdminPermissionName,
		type AdminRole
	} from '#lib/api/index.js';
	import {
		Alert,
		Badge,
		Button,
		FullscreenDialog,
		FormSection,
		Input,
		Textarea,
		notify
	} from '#lib/components/ui/index.js';
	import Choice from '#lib/components/roles/Choice.svelte';
	import { tidyName } from '#lib/components/roles/roles.js';
	import { keys } from '#lib/query/index.js';
	import { submit } from '#lib/forms.js';
	import { byGroup } from './permissions';

	type Props = {
		role: AdminRole | null;
		catalog: AdminPermission[];
		open: boolean;
	};

	let { role, catalog, open = $bindable(false) }: Props = $props();

	const queryClient = useQueryClient();

	let name = $state('');
	let description = $state('');
	let granted = $state<AdminPermissionName[]>([]);

	let saving = $state(false);

	const editing = $derived(role !== null);

	const locked = $derived(role?.is_builtin ?? false);

	const groups = $derived(byGroup(catalog));

	$effect(() => {
		if (!open) return;

		name = role?.name ?? '';
		description = role?.description ?? '';
		granted = role ? [...role.permissions] : [];
	});

	function toggle(permission: AdminPermissionName, on: boolean) {
		granted = on ? [...granted, permission] : granted.filter((it) => it !== permission);
	}

	function payload() {
		return { name: tidyName(name), description: description.trim(), permissions: granted };
	}

	const save = submit<Awaited<ReturnType<typeof adminsApi.updateRole>>>({
		input: () => (role ? [role.id, payload()] : [payload()]),
		when: () => !(locked || name.trim() === ''),
		pending: (busy) => (saving = busy),
		done: async () => {
			notify.success(role ? 'Admin role saved' : 'Admin role created');
			await queryClient.invalidateQueries({ queryKey: keys.admins.all });
			open = false;
		},
		failed: (err) => notify.error(err, 'Could not save this role')
	});
</script>

{#snippet actions()}
	<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>
	<Button type="submit" loading={saving} disabled={saving || name.trim() === ''}>
		{saving ? 'Saving…' : editing ? 'Save changes' : 'Create role'}
	</Button>
{/snippet}

<FullscreenDialog
	bind:open
	actions={!locked ? actions : undefined}
	title={role ? role.name : 'New admin role'}
	meta={role
		? `${role.admin_count} ${role.admin_count === 1 ? 'administrator' : 'administrators'}`
		: undefined}
	action={role ? '/actions/admins?/updateRole' : '/actions/admins?/createRole'}
	onsubmit={save}
>
	{#if locked}
		<div class="intro">
			<Alert tone="info">
				super_admin grants every permission, and is the only role that can manage administrators and
				admin roles. It cannot be changed or removed.
			</Alert>
		</div>
	{/if}

	<fieldset disabled={locked}>
		<FormSection title="Role" description="What administrators holding this role are called.">
			<Input
				label="Name"
				icon="ri-shield-keyhole-line"
				bind:value={name}
				autocomplete="off"
				autocapitalize="none"
				spellcheck={false}
				placeholder="moderator"
				hint="Lower case letters, numbers, dashes and underscores."
				required
			/>

			<Textarea
				label="Description"
				bind:value={description}
				rows={2}
				placeholder="What administrators holding this role look after"
			/>

			{#if editing}
				<Input label="Role ID" value={role?.id ?? ''} readOnly copyable />
			{/if}
		</FormSection>

		<FormSection
			title="Permissions"
			description="What administrators holding this role may do. Held for one application, the role grants only the permissions marked per application, and only there. Managing administrators stays with super_admin."
		>
			{#each groups as [group, permissions] (group)}
				<div class="group">
					<span class="label">{group}</span>

					<div class="list">
						{#each permissions as permission (permission.name)}
							<Choice
								name={permission.name}
								description={permission.description}
								checked={granted.includes(permission.name)}
								onChange={(on) => toggle(permission.name, on)}
								disabled={locked}
							>
								{#snippet badges()}
									{#if permission.scopable}
										<Badge tone="success">per application</Badge>
									{/if}
									{#if permission.name.endsWith('.write')}
										<Badge>write</Badge>
									{/if}
								{/snippet}
							</Choice>
						{/each}
					</div>
				</div>
			{/each}
		</FormSection>
	</fieldset>
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

	.group + .group {
		margin-top: var(--space-2);
	}

	.group {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.label {
		padding-left: var(--space-2);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.list {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}
</style>
