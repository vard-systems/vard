<script lang="ts">
	import {
		RiKey2Line,
		RiMailLine,
		RiShieldKeyholeLine,
		RiText,
		RiTimeLine,
		RiToggleLine
	} from '#lib/icons.js';
	import type { AdminRecord, AdminStatus } from '#lib/api/index.js';
	import { Badge, Code, type Column, DataTable, Tag } from '#lib/components/ui/index.js';
	import { formatDateTime } from '#lib/utils/format.js';

	type Props = {
		admins: AdminRecord[];
		self: string;
		onOpen: (admin: AdminRecord) => void;
		selected: string[];
		onSelect: (ids: string[]) => void;
	};

	let { admins, self, onOpen, selected, onSelect }: Props = $props();

	const columns: Column[] = [
		{ key: 'id', label: 'ID', icon: RiKey2Line, min: '10rem' },
		{ key: 'email', label: 'Email', icon: RiMailLine, min: '14rem' },
		{ key: 'name', label: 'Name', icon: RiText, min: '10rem' },
		{ key: 'status', label: 'Status', icon: RiToggleLine, min: '8rem' },
		{ key: 'mfa', label: 'Two-factor', icon: RiShieldKeyholeLine, min: '8rem' },
		{ key: 'roles', label: 'Roles', icon: RiShieldKeyholeLine, min: '12rem' },
		{ key: 'last_login_at', label: 'Last sign-in', icon: RiTimeLine, min: '11rem' }
	];

	const tones: Record<AdminStatus, 'success' | 'danger' | 'neutral'> = {
		active: 'success',
		suspended: 'danger',
		disabled: 'danger',
		invited: 'neutral'
	};

	function shortId(id: string): string {
		return id.replace(/-/g, '').slice(0, 15);
	}
</script>

<DataTable
	{columns}
	rows={admins}
	empty="No administrators match this."
	{onOpen}
	label={(admin) => `Edit ${admin.email}`}
	{selected}
	{onSelect}
>
	{#snippet row(admin)}
		<td><Code title={admin.id}>{shortId(admin.id)}</Code></td>

		<td>
			<span class="text">{admin.email}</span>
			{#if admin.id === self}
				<Badge>you</Badge>
			{/if}
		</td>

		<td><span class="text">{admin.full_name}</span></td>

		<td><Badge tone={tones[admin.status]}>{admin.status}</Badge></td>

		<td>
			{#if admin.mfa_enabled}
				<Tag tone="success" dot strong>on</Tag>
			{:else}
				<Tag dot>off</Tag>
			{/if}
		</td>

		<td>
			<span class="badges">
				{#each admin.assignments as assignment (assignment.id)}
					<Badge tone={assignment.role.name === 'super_admin' ? 'success' : 'neutral'}>
						{assignment.role.name}
						{#if assignment.application}
							<span class="scope">in {assignment.application.name}</span>
						{/if}
					</Badge>
				{:else}
					<span class="empty">N/A</span>
				{/each}
			</span>
		</td>

		<td>
			{#if admin.last_login_at}
				<span class="mono">{formatDateTime(admin.last_login_at)}</span>
			{:else}
				<span class="empty">never</span>
			{/if}
		</td>
	{/snippet}
</DataTable>

<style>
	.text {
		margin-right: var(--space-1);
	}

	.badges {
		display: inline-flex;
		gap: var(--space-1);
	}

	.scope {
		margin-left: 4px;
		opacity: 0.65;
	}

	.mono {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.empty {
		color: var(--color-text-disabled);
		font-size: var(--text-sm);
	}
</style>
