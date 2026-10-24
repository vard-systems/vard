<script lang="ts">
	import type { AdminPermission, AdminRole, Application } from '#lib/api/index.js';
	import {
		Badge,
		Button,
		FormSection,
		IconButton,
		Select,
		type SelectOption
	} from '#lib/components/ui/index.js';
	import { PANEL, withRole, type Row } from './assignments';
	import { grantedBy } from './permissions';

	type Props = {
		rows: Row[];
		roles: AdminRole[];
		applications: Application[];
		catalog: AdminPermission[];
		/** Editing yourself: super_admin for the whole panel cannot be taken away. */
		isSelf: boolean;
		/** A fresh key for a new row. */
		nextKey: () => number;
	};

	let { rows = $bindable([]), roles, applications, catalog, isSelf, nextKey }: Props = $props();

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

	/** What a role held for one application grants that only counts for the whole panel, worth
	    pointing out: it will not apply. */
	function unscoped(row: Row): string[] {
		if (row.scope === PANEL) return [];

		const role = roles.find((it) => it.id === row.role);
		const scopable = new Set(catalog.filter((it) => it.scopable).map((it) => it.name));

		return (role?.permissions ?? []).filter((name) => !scopable.has(name));
	}

	function locked(row: Row): boolean {
		return isSelf && row.role === superRole?.id && row.scope === PANEL;
	}

	function setScope(key: number, scope: string) {
		rows = rows.map((row) => (row.key === key ? { ...row, scope } : row));
	}
</script>

<FormSection
	title="Roles"
	description="Each role is held for the whole panel or for one application. A role held for one application grants only its application permissions, and only there."
>
	<div class="rows">
		{#each rows as row (row.key)}
			{@const ignored = unscoped(row)}
			<div class="row">
				<Select
					label="Role"
					value={row.role}
					options={roleOptions}
					placeholder="Choose a role"
					readOnly={locked(row)}
					onChange={(value) => (rows = withRole(rows, row.key, value, superRole?.id))}
				/>

				<Select
					label="For"
					value={row.scope}
					options={scopeOptions}
					readOnly={locked(row) || row.role === superRole?.id}
					onChange={(value) => setScope(row.key, value)}
				/>

				<IconButton
					icon="ri-close-line"
					label="Remove this role"
					onclick={() => (rows = rows.filter((it) => it.key !== row.key))}
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
		<Button
			size="sm"
			variant="subtle"
			onclick={() => (rows = [...rows, { key: nextKey(), role: '', scope: PANEL }])}
		>
			<i class="ri-add-line" aria-hidden="true"></i>
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

<style>
	.hint {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.4;
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
		.row,
		.summary {
			grid-template-columns: 1fr;
		}
	}
</style>
