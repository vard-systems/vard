<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import {
		RiAddLine,
		RiArrowLeftLine,
		RiCloseLine,
		RiGlobalLine,
		RiNodeTree,
		RiShieldUserLine
	} from '#lib/icons.js';
	import {
		usersApi,
		type Admin,
		type Application,
		type Role,
		type RoleMapping
	} from '#lib/api/index.js';
	import {
		Alert,
		Button,
		Icon,
		IconButton,
		Switch,
		Tooltip,
		notify
	} from '#lib/components/ui/index.js';
	import { canAnywhere } from '#lib/permissions.js';
	import { keys, roleMappingsOptions } from '#lib/query/index.js';
	import { countOf } from '#lib/utils/format.js';
	import RolePicker from './RolePicker.svelte';
	import { canAssign } from './roles';

	type Props = {
		userId: string;
		roles: Role[];
		applications: Application[];
		admin: Admin;
	};

	let { userId, roles, applications, admin }: Props = $props();

	const queryClient = useQueryClient();

	const mappings = createQuery(() => roleMappingsOptions(userId));

	let mode = $state<'held' | 'assign'>('held');

	let showInherited = $state(false);
	let picked = $state<string[]>([]);

	let removing = $state<string | null>(null);

	let busy = $state(false);

	const mayAssign = $derived(canAnywhere(admin, 'role_assignments.write'));

	const held = $derived(mappings.data ?? []);
	const assigned = $derived(held.filter((it) => it.assigned));
	const inherited = $derived(held.filter((it) => !it.assigned));

	const rows = $derived.by(() => {
		const out: { key: string; name: string; clientId?: string; roles: RoleMapping[] }[] = [];

		for (const mapping of held) {
			if (!mapping.assigned && !showInherited) continue;

			const key = mapping.application?.id ?? 'global';
			let row = out.find((it) => it.key === key);

			if (!row) {
				row = {
					key,
					name: mapping.application?.name ?? 'Global',
					clientId: mapping.application?.client_id,
					roles: []
				};
				out.push(row);
			}

			row.roles.push(mapping);
		}

		for (const row of out) {
			row.roles.sort((a, b) => Number(b.assigned) - Number(a.assigned));
		}

		return out;
	});

	function explain(role: RoleMapping): string {
		const parts = [role.description];

		if (!role.assigned) {
			parts.push(`Comes with ${role.via.map((it) => it.name).join(', ')}`);
		} else {
			const gives = held.filter((it) => it.via.some((through) => through.id === role.id));
			if (gives.length > 0) {
				parts.push(
					`Also gives ${gives.map((it) => (it.application ? `${it.application.name} / ${it.name}` : it.name)).join(', ')}`
				);
			}
		}

		return parts.filter(Boolean).join(' · ') || role.name;
	}

	const assignable = $derived(roles.filter((role) => !assigned.some((it) => it.id === role.id)));

	function scopeOf(mapping: RoleMapping) {
		return { application_id: mapping.application?.id ?? null };
	}

	async function refill(next: RoleMapping[]) {
		queryClient.setQueryData(keys.users.mappings(userId), next);
		await Promise.all([
			queryClient.invalidateQueries({ queryKey: ['users', 'list'] }),
			queryClient.invalidateQueries({ queryKey: keys.roles.all })
		]);
	}

	const assign = createMutation(() => ({
		mutationFn: (ids: string[]) => usersApi.assignRoles(userId, ids),
		onSuccess: async (result: { roles: RoleMapping[] }, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'role')} assigned`);
			await refill(result.roles);
			picked = [];
			mode = 'held';
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not assign these roles');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	const unassign = createMutation(() => ({
		mutationFn: (id: string) => usersApi.unassignRole(userId, id),
		onSuccess: (result: { roles: RoleMapping[] }) => {
			notify.success('Role removed');
			return refill(result.roles);
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not remove this role');
		},
		onSettled: () => {
			busy = false;
			removing = null;
		}
	}));

	function remove(role: RoleMapping) {
		if (busy) return;

		busy = true;
		removing = role.id;
		unassign.mutate(role.id);
	}

	function openAssign() {
		picked = [];
		mode = 'assign';
	}

	function submitAssign() {
		if (busy || picked.length === 0) return;

		busy = true;
		assign.mutate(picked);
	}
</script>

{#if mode === 'held'}
	<div class="bar">
		<p class="summary">
			{#if mappings.isSuccess}
				<strong>{assigned.length}</strong>
				{assigned.length === 1 ? 'role' : 'roles'}
				{#if inherited.length > 0}
					<span class="dot">·</span>
					<strong>{inherited.length}</strong> inherited
				{/if}
			{/if}
		</p>

		{#if inherited.length > 0}
			<Switch label="Show inherited" bind:checked={showInherited} />
		{/if}

		{#if mayAssign}
			<Button size="sm" onclick={openAssign} disabled={busy}>
				<Icon icon={RiAddLine} />
				Assign roles
			</Button>
		{/if}
	</div>

	{#if mappings.isPending}
		<div class="panel loading" aria-busy="true">
			{#each [0, 1] as row (row)}
				<div class="row">
					<span class="skeleton label-skeleton"></span>
					<span class="skeleton chip-skeleton"></span>
				</div>
			{/each}
		</div>
	{:else if mappings.isError}
		<Alert>Could not load this user's roles.</Alert>
	{:else if held.length === 0}
		<div class="empty">
			<Icon icon={RiShieldUserLine} size="1.25rem" />
			<div>
				<strong>No roles yet</strong>
				<p>Roles decide what this user may do in your applications.</p>
			</div>
		</div>
	{:else}
		<div class="panel">
			{#each rows as row (row.key)}
				<div class="row">
					<div class="scope" title={row.clientId}>
						<Icon icon={row.key === 'global' ? RiGlobalLine : RiShieldUserLine} size="0.9375rem" />
						<span>{row.name}</span>
					</div>

					<ul class="chips">
						{#each row.roles as role (role.id)}
							{@const removable = role.assigned && canAssign(admin, scopeOf(role))}
							<li>
								<Tooltip label={explain(role)} placement="top">
									{#snippet children(trigger)}
										<span
											{...trigger()}
											class="chip"
											class:inherited={!role.assigned}
											class:removable
											class:leaving={removing === role.id}
										>
											{#if role.composite}
												<Icon icon={RiNodeTree} size="0.8125rem" />
											{/if}
											{role.name}
											{#if removable}
												<button
													type="button"
													class="remove"
													aria-label="Remove {role.name}"
													onclick={() => remove(role)}
													disabled={busy}
												>
													<Icon icon={RiCloseLine} size="0.875rem" />
												</button>
											{/if}
										</span>
									{/snippet}
								</Tooltip>
							</li>
						{/each}
					</ul>
				</div>
			{:else}
				<p class="none">
					No roles were given directly. Turn on <em>Show inherited</em> to see the
					{inherited.length} that come with other roles.
				</p>
			{/each}
		</div>

		<div class="legend">
			<span><span class="swatch"></span>Assigned</span>
			{#if inherited.length > 0}
				<span><span class="swatch inherited"></span>Inherited — comes with a composite role</span>
			{/if}
			<span><Icon icon={RiNodeTree} size="0.8125rem" />Composite — includes other roles</span>
		</div>
	{/if}
{:else}
	<div class="assign-head">
		<IconButton
			icon={RiArrowLeftLine}
			label="Back to the user's roles"
			size="sm"
			onclick={() => (mode = 'held')}
			disabled={busy}
		/>
		<div>
			<strong>Assign roles</strong>
			<p>Pick any number of roles, global or from any application.</p>
		</div>
	</div>

	<RolePicker
		roles={assignable}
		{applications}
		bind:picked
		reason={(role) =>
			canAssign(admin, role) ? undefined : 'Your roles do not allow assigning this role.'}
		empty="This user already has every role there is."
	/>

	<div class="assign-foot">
		<span class="hint">
			{picked.length === 0
				? 'Nothing selected'
				: `${picked.length} ${picked.length === 1 ? 'role' : 'roles'} selected`}
		</span>
		<Button variant="subtle" size="sm" onclick={() => (mode = 'held')} disabled={busy}>
			Cancel
		</Button>
		<Button size="sm" onclick={submitAssign} disabled={busy || picked.length === 0} loading={busy}>
			Assign{picked.length > 0 ? ` ${picked.length}` : ''}
		</Button>
	</div>
{/if}

<style>
	.bar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2) var(--space-4);
		margin-bottom: var(--space-3);
	}

	.summary {
		flex: 1;
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-base);
		white-space: nowrap;
	}

	.summary strong {
		color: var(--color-text);
	}

	.dot {
		margin: 0 4px;
	}

	.panel {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.row {
		display: grid;
		grid-template-columns: 9rem 1fr;
		align-items: start;
		gap: var(--space-3);
		padding: 10px var(--space-3);
	}

	.row + .row {
		border-top: 1px solid var(--color-border);
	}

	.scope {
		display: flex;
		align-items: center;
		gap: 6px;
		min-width: 0;
		height: 28px;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.scope span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.chip {
		display: inline-flex;
		align-items: center;
		gap: 5px;
		height: 28px;
		padding: 0 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 500;
		white-space: nowrap;
		cursor: default;
		transition: opacity var(--speed-fast);
	}

	.chip.removable {
		padding-right: 3px;
	}

	.chip.inherited {
		border-style: dashed;
		background: transparent;
		color: var(--color-text-hint);
	}

	.chip.leaving {
		opacity: 0.45;
	}

	.remove {
		display: grid;
		place-items: center;
		width: 22px;
		height: 22px;
		padding: 0;
		border: none;
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--color-text-hint);
		cursor: pointer;
		transition:
			background-color var(--speed-fast),
			color var(--speed-fast);
	}

	.remove:hover:not(:disabled) {
		background: var(--surface-danger);
		color: var(--color-danger);
	}

	.remove:focus-visible {
		outline: 2px solid var(--color-info);
	}

	.none {
		margin: 0;
		padding: var(--space-3);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1) var(--space-4);
		margin-top: var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.legend > span {
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}

	.swatch {
		width: 16px;
		height: 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
	}

	.swatch.inherited {
		border-style: dashed;
		background: transparent;
	}

	.empty {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-4);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-surface);
		color: var(--color-text-hint);
	}

	.empty strong {
		color: var(--color-text);
	}

	.empty p {
		margin: 2px 0 0;
		font-size: var(--text-sm);
	}

	.skeleton {
		display: block;
		height: 28px;
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		animation: pulse 1.2s ease-in-out infinite;
	}

	.label-skeleton {
		width: 6rem;
	}

	.chip-skeleton {
		width: 14rem;
	}

	@keyframes pulse {
		50% {
			opacity: 0.5;
		}
	}

	.assign-head {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.assign-head p {
		margin: 2px 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.assign-foot {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: var(--space-2);
		margin-top: var(--space-4);
		padding-top: var(--space-3);
		border-top: 1px solid var(--color-border);
	}

	.hint {
		flex: 1;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	@media (max-width: 34rem) {
		.row {
			grid-template-columns: 1fr;
			gap: var(--space-2);
		}

		.scope {
			height: auto;
		}

		.summary {
			flex-basis: 100%;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.skeleton {
			animation: none;
		}
	}
</style>
