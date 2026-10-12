<script lang="ts">
	import { resolve } from '$app/paths';
	import {
		RiAdminLine,
		RiFileTextLine,
		RiKey2Line,
		RiLockLine,
		RiShieldKeyholeLine
	} from 'svelte-remixicon';
	import type { AdminRole } from '$lib/api';
	import { Badge, Code, type Column, DataTable } from '$lib/components/ui';

	type Props = {
		roles: AdminRole[];
		onOpen: (role: AdminRole) => void;
		selected: string[];
		onSelect: (ids: string[]) => void;
	};

	let { roles, onOpen, selected, onSelect }: Props = $props();

	const columns: Column[] = [
		{ key: 'id', label: 'ID', icon: RiKey2Line, min: '10rem' },
		{ key: 'name', label: 'Name', icon: RiShieldKeyholeLine, min: '10rem' },
		{ key: 'description', label: 'Description', icon: RiFileTextLine, min: '14rem' },
		{ key: 'permissions', label: 'Permissions', icon: RiLockLine, min: '16rem' },
		{ key: 'admins', label: 'Administrators', icon: RiAdminLine, min: '8rem' }
	];

	const SHOWN = 3;

	function shortId(id: string): string {
		return id.replace(/-/g, '').slice(0, 15);
	}

	function holders(role: AdminRole): string {
		return `${resolve('/admin/(panel)/dashboard/admins')}?role=${role.id}`;
	}
</script>

<DataTable
	{columns}
	rows={roles}
	empty="No admin roles match this."
	{onOpen}
	label={(role) => `Open ${role.name}`}
	{selected}
	{onSelect}
>
	{#snippet row(role)}
		<td><Code title={role.id}>{shortId(role.id)}</Code></td>

		<td>
			<span class="name">{role.name}</span>
			{#if role.is_builtin}
				<Badge tone="success">built in</Badge>
			{/if}
		</td>

		<td>
			{#if role.description}
				<span class="text">{role.description}</span>
			{:else}
				<span class="empty">N/A</span>
			{/if}
		</td>

		<td>
			<span class="badges">
				{#if role.is_builtin}
					<Badge tone="success">everything</Badge>
				{:else}
					{#each role.permissions.slice(0, SHOWN) as permission (permission)}
						<Badge>{permission}</Badge>
					{:else}
						<span class="empty">none</span>
					{/each}
					{#if role.permissions.length > SHOWN}
						<span class="more">+{role.permissions.length - SHOWN} more</span>
					{/if}
				{/if}
			</span>
		</td>

		<td>
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="count" href={holders(role)} onclick={(event) => event.stopPropagation()}>
				{role.admin_count}
			</a>
		</td>
	{/snippet}
</DataTable>

<style>
	.name {
		margin-right: var(--space-1);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.text {
		display: inline-block;
		max-width: 22rem;
		overflow: hidden;
		text-overflow: ellipsis;
		vertical-align: middle;
		white-space: nowrap;
	}

	.badges {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}

	.empty {
		color: var(--color-text-disabled);
		font-size: var(--text-sm);
	}

	.more {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.count {
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		text-decoration: underline;
		text-decoration-color: var(--color-border);
		text-underline-offset: 3px;
	}

	.count:hover {
		text-decoration-color: currentColor;
	}
</style>
