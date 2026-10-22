<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Application } from '#lib/api/index.js';
	import { Badge, CellText, Code, type Column, DataTable, Tag } from '#lib/components/ui/index.js';
	import { ADMIN_CLI, types } from './applications';

	type Props = {
		applications: Application[];
		onOpen: (application: Application) => void;
		selected?: string[];
		onSelect?: (ids: string[]) => void;
	};

	let { applications, onOpen, selected, onSelect }: Props = $props();

	const columns: Column[] = [
		{ key: 'name', label: 'Name', min: '14rem', grow: true },
		{ key: 'type', label: 'Type', min: '10rem' },
		{ key: 'client_id', label: 'Client ID', min: '12rem' },
		{ key: 'redirect_uris', label: 'Redirect URIs', min: '14rem' },
		{ key: 'roles', label: 'Roles', min: '5rem', align: 'end' },
		{ key: 'status', label: 'Status', min: '7rem' }
	];

	function roles(application: Application): string {
		return `${resolve('/(panel)/roles')}?application=${application.id}`;
	}
</script>

<DataTable
	{columns}
	rows={applications}
	empty="No applications match this."
	{onOpen}
	label={(application) => `Open ${application.name}`}
	{selected}
	{onSelect}
>
	{#snippet row(application)}
		{@const kind = types[application.type]}
		<td>
			<CellText title={application.name} detail={application.description || undefined}>
				{#if application.client_id === ADMIN_CLI}
					<Badge>Built-in</Badge>
				{/if}
			</CellText>
		</td>

		<td>
			<span class="type">
				<i class={kind.icon} aria-hidden="true"></i>
				{kind.label}
			</span>
		</td>

		<td>
			<span class="client">
				<Code tone="quiet" truncate title={application.client_id}>{application.client_id}</Code>
			</span>
		</td>

		<td>
			{#if application.redirect_uris.length > 0}
				<span class="uris">
					<span class="uri" title={application.redirect_uris.join('\n')}>
						{application.redirect_uris[0]}
					</span>
					{#if application.redirect_uris.length > 1}
						<span class="more">+{application.redirect_uris.length - 1}</span>
					{/if}
				</span>
			{:else}
				<span class="empty">—</span>
			{/if}
		</td>

		<td class="end">
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="count" href={roles(application)} onclick={(event) => event.stopPropagation()}>
				{application.role_count}
			</a>
		</td>

		<td>
			<Tag tone={application.is_enabled ? 'success' : 'neutral'} dot>
				{application.is_enabled ? 'Enabled' : 'Disabled'}
			</Tag>
		</td>
	{/snippet}
</DataTable>

<style>
	.type {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--color-text-hint);
	}

	.client {
		display: inline-flex;
		max-width: 12rem;
	}

	.uris {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}

	.uri {
		display: inline-block;
		max-width: 16rem;
		overflow: hidden;
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.more {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.empty {
		color: var(--color-text-disabled);
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
