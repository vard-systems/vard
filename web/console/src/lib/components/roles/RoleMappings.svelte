<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import {
		usersApi,
		type Admin,
		type Application,
		type Role,
		type RoleMapping
	} from '#lib/api/index.js';
	import { Alert, Button, EmptyState, Switch, notify } from '#lib/components/ui/index.js';
	import { canAnywhere } from '#lib/permissions.js';
	import { keys, roleMappingsOptions } from '#lib/query/index.js';
	import { countOf } from '#lib/utils/format.js';
	import AssignRoles from './AssignRoles.svelte';
	import HeldRoles from './HeldRoles.svelte';
	import { byScope } from './mappings';

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
	let removing = $state<string | null>(null);
	let busy = $state(false);

	const mayAssign = $derived(canAnywhere(admin, 'role_assignments.write'));

	const held = $derived(mappings.data ?? []);
	const assigned = $derived(held.filter((it) => it.assigned));
	const inherited = $derived(held.filter((it) => !it.assigned));
	const rows = $derived(byScope(held, showInherited));
	const assignable = $derived(roles.filter((role) => !assigned.some((it) => it.id === role.id)));

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

	function submitAssign(ids: string[]) {
		if (busy || ids.length === 0) return;

		busy = true;
		assign.mutate(ids);
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
			<Button size="sm" onclick={() => (mode = 'assign')} disabled={busy}>
				<i class="ri-add-line" aria-hidden="true"></i>
				Assign roles
			</Button>
		{/if}
	</div>

	{#if mappings.isPending}
		<div class="loading" aria-busy="true">
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
		<EmptyState icon="ri-shield-user-line" title="No roles yet">
			Roles decide what this user may do in your applications.
		</EmptyState>
	{:else}
		<HeldRoles
			{rows}
			{held}
			inherited={inherited.length}
			{admin}
			{busy}
			{removing}
			onremove={remove}
		/>
	{/if}
{:else}
	<AssignRoles
		roles={assignable}
		{applications}
		{admin}
		{busy}
		onassign={submitAssign}
		oncancel={() => (mode = 'held')}
	/>
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

	.loading {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.row {
		display: grid;
		grid-template-columns: 9rem 1fr;
		gap: var(--space-3);
		padding: 10px var(--space-3);
	}

	.row + .row {
		border-top: 1px solid var(--color-border);
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

	@media (max-width: 34rem) {
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
