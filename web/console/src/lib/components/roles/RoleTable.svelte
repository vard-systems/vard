<script lang="ts">
	import { resolve } from '$app/paths';
	import {
		RiAppsLine,
		RiFileTextLine,
		RiGroupLine,
		RiNodeTree,
		RiShieldUserLine
	} from 'svelte-remixicon';
	import type { Application, Role, UserRoleRef } from '#lib/api/index.js';
	import { Badge, DataTable, Icon, type Column } from '#lib/components/ui/index.js';

	type Props = {
		roles: Role[];
		showApplication?: boolean;
		applications: Application[];
		onOpen: (role: Role) => void;
		selected?: string[];
		onSelect?: (ids: string[]) => void;
	};

	let {
		roles,
		showApplication = false,
		applications,
		onOpen,
		selected,
		onSelect
	}: Props = $props();

	function label(included: UserRoleRef, from: Role): string {
		if (included.application_id === from.application_id) return included.name;
		if (included.application_id === null) return `Global / ${included.name}`;

		const app = applications.find((it) => it.id === included.application_id);
		return `${app?.name ?? '?'} / ${included.name}`;
	}

	const columns = $derived<Column[]>([
		{ key: 'name', label: 'Role', icon: RiShieldUserLine, min: '12rem' },
		...(showApplication
			? [{ key: 'application', label: 'Application', icon: RiAppsLine, min: '10rem' }]
			: []),
		{ key: 'description', label: 'Description', icon: RiFileTextLine, min: '16rem' },
		{ key: 'includes', label: 'Includes', icon: RiNodeTree, min: '12rem' },
		{ key: 'users', label: 'Users', icon: RiGroupLine, min: '6rem', align: 'end' }
	]);

	function appName(role: Role): string {
		return applications.find((app) => app.id === role.application_id)?.name ?? '?';
	}

	const SHOWN = 3;

	function holders(role: Role): string {
		return `${resolve('/admin/(panel)/dashboard/users')}?role=${role.id}`;
	}
</script>

<DataTable
	{columns}
	rows={roles}
	empty="No roles match this."
	{onOpen}
	label={(role) => `Edit ${role.name}`}
	{selected}
	{onSelect}
>
	{#snippet row(role)}
		<td>
			<span class="name">{role.name}</span>
			{#if role.is_default}
				<Badge tone="success">default</Badge>
			{/if}
		</td>

		{#if showApplication}
			<td>
				<span class="app">
					<Icon icon={RiShieldUserLine} size="0.9375rem" />
					{appName(role)}
				</span>
			</td>
		{/if}

		<td>
			{#if role.description}
				<span class="text">{role.description}</span>
			{:else}
				<span class="empty">—</span>
			{/if}
		</td>

		<td>
			<span class="badges">
				{#each role.inherited_roles.slice(0, SHOWN) as inherited (inherited.id)}
					<Badge>{label(inherited, role)}</Badge>
				{:else}
					<span class="empty">—</span>
				{/each}
				{#if role.inherited_roles.length > SHOWN}
					<span class="more">+{role.inherited_roles.length - SHOWN} more</span>
				{/if}
			</span>
		</td>

		<td class="end">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="count" href={holders(role)} onclick={(event) => event.stopPropagation()}>
				{role.user_count}
			</a>
		</td>
	{/snippet}
</DataTable>

<style>
	.app {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		color: var(--color-text);
	}

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

	.empty,
	.more {
		color: var(--color-text-disabled);
		font-size: var(--text-sm);
	}

	.more {
		color: var(--color-text-hint);
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
