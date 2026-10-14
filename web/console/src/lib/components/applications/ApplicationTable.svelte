<script lang="ts">
	import { resolve } from '$app/paths';
	import {
		RiKey2Line,
		RiLinksLine,
		RiShieldUserLine,
		RiText,
		RiToggleLine,
		RiShapesLine
	} from 'svelte-remixicon';
	import type { Application } from '#lib/api/index.js';
	import { Badge, Code, type Column, DataTable, Icon } from '#lib/components/ui/index.js';
	import { ADMIN_CLI, types } from './applications';

	type Props = {
		applications: Application[];
		onOpen: (application: Application) => void;
		selected?: string[];
		onSelect?: (ids: string[]) => void;
	};

	let { applications, onOpen, selected, onSelect }: Props = $props();

	const columns: Column[] = [
		{ key: 'name', label: 'Name', icon: RiText, min: '12rem' },
		{ key: 'type', label: 'Type', icon: RiShapesLine, min: '11rem' },
		{ key: 'client_id', label: 'Client ID', icon: RiKey2Line, min: '17rem' },
		{ key: 'redirect_uris', label: 'Redirect URIs', icon: RiLinksLine, min: '14rem' },
		{ key: 'roles', label: 'Roles', icon: RiShieldUserLine, min: '6rem' },
		{ key: 'is_enabled', label: 'Is Enabled', icon: RiToggleLine, min: '7rem' }
	];

	function roles(application: Application): string {
		return `${resolve('/admin/(panel)/dashboard/roles')}?application=${application.id}`;
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
		<td>
			<span class="name">
				{application.name}
				{#if application.client_id === ADMIN_CLI}
					<Badge>Built-in</Badge>
				{/if}
			</span>
			{#if application.description}
				<span class="hint">{application.description}</span>
			{/if}
		</td>

		<td>
			<span class="type">
				<Icon icon={types[application.type].icon} />
				{types[application.type].label}
			</span>
		</td>

		<td><Code>{application.client_id}</Code></td>

		<td>
			{#if application.redirect_uris.length > 0}
				<span class="uri">{application.redirect_uris[0]}</span>
				{#if application.redirect_uris.length > 1}
					<span class="hint">+{application.redirect_uris.length - 1}</span>
				{/if}
			{:else}
				<span class="empty">N/A</span>
			{/if}
		</td>

		<td>
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="count" href={roles(application)} onclick={(event) => event.stopPropagation()}>
				{application.role_count}
			</a>
		</td>

		<td>
			<Badge tone={application.is_enabled ? 'success' : 'neutral'}>
				{application.is_enabled ? 'True' : 'False'}
			</Badge>
		</td>
	{/snippet}
</DataTable>

<style>
	.name {
		margin-right: var(--space-2);
		font-weight: 600;
	}

	.hint,
	.empty {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.empty {
		color: var(--color-text-disabled);
	}

	.type {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}

	.uri {
		display: inline-block;
		max-width: 18rem;
		overflow: hidden;
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		text-overflow: ellipsis;
		vertical-align: middle;
		white-space: nowrap;
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
