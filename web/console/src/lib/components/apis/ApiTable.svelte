<script lang="ts">
	import {
		RiAppsLine,
		RiCodeBoxLine,
		RiLinksLine,
		RiLockLine,
		RiShieldCheckLine
	} from 'svelte-remixicon';
	import type { API } from '$lib/api';
	import { Badge, DataTable, type Column } from '$lib/components/ui';

	type Props = {
		apis: API[];
		onOpen: (api: API) => void;
		selected?: string[];
		onSelect?: (ids: string[]) => void;
	};

	let { apis, onOpen, selected, onSelect }: Props = $props();

	const columns: Column[] = [
		{ key: 'name', label: 'API', icon: RiCodeBoxLine, min: '12rem' },
		{ key: 'identifier', label: 'Identifier', icon: RiLinksLine, min: '16rem' },
		{ key: 'scopes', label: 'Scopes', icon: RiLockLine, min: '16rem' },
		{ key: 'access', label: 'Role-based access', icon: RiShieldCheckLine, min: '10rem' },
		{ key: 'applications', label: 'Applications', icon: RiAppsLine, min: '8rem', align: 'end' }
	];

	const SHOWN = 3;
</script>

<DataTable
	{columns}
	rows={apis}
	empty="No APIs yet. Register one to give applications something to ask for tokens for."
	{onOpen}
	label={(api) => `Open ${api.name}`}
	{selected}
	{onSelect}
>
	{#snippet row(api)}
		<td>
			<span class="name">
				{api.name}
				{#if api.system}
					<Badge>Built-in</Badge>
				{/if}
			</span>
			{#if api.description}
				<span class="muted">{api.description}</span>
			{/if}
		</td>

		<td><code>{api.identifier}</code></td>

		<td>
			<span class="badges">
				{#each api.scopes.slice(0, SHOWN) as scope (scope.id)}
					<Badge>{scope.name}</Badge>
				{:else}
					<span class="muted">—</span>
				{/each}
				{#if api.scopes.length > SHOWN}
					<span class="muted">+{api.scopes.length - SHOWN} more</span>
				{/if}
			</span>
		</td>

		<td>
			<Badge tone={api.enforce_roles ? 'success' : 'neutral'}>
				{api.enforce_roles ? 'Enforced' : 'Off'}
			</Badge>
		</td>

		<td class="end"><span class="count">{api.application_count}</span></td>
	{/snippet}
</DataTable>

<style>
	.name {
		margin-right: var(--space-2);
		font-weight: 600;
	}

	.muted {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	code {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.badges {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}

	.count {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}
</style>
