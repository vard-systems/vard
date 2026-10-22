<script lang="ts">
	import type { CacheKey } from '#lib/api/index.js';
	import { Code, type Column, DataTable, Tag } from '#lib/components/ui/index.js';
	import { formatBytes, formatTTL, kinds } from './cache';

	type Props = {
		keys: CacheKey[];
		empty: string;
		onOpen: (key: CacheKey) => void;
	};

	let { keys, empty, onOpen }: Props = $props();

	const columns: Column[] = [
		{ key: 'name', label: 'Key', min: '20rem', grow: true },
		{ key: 'kind', label: 'Kind', min: '8rem' },
		{ key: 'group', label: 'Group', min: '9rem' },
		{ key: 'ttl', label: 'Expires in', min: '8rem', align: 'end' },
		{ key: 'size', label: 'Size', min: '6rem', align: 'end' }
	];

	const rows = $derived(keys.map((key) => ({ ...key, id: key.name })));
</script>

<DataTable {columns} {rows} {empty} {onOpen} label={(key) => `Open ${key.name}`}>
	{#snippet row(key)}
		<td><Code truncate title={key.name}>{key.name}</Code></td>
		<td><Tag small tone={kinds[key.kind].tone}>{kinds[key.kind].label}</Tag></td>
		<td>
			{#if key.group}
				<span class="mono">{key.group}</span>
			{:else}
				<span class="empty">N/A</span>
			{/if}
		</td>
		<td class="number">{formatTTL(key.ttl_seconds)}</td>
		<td class="number">{key.size_bytes > 0 ? formatBytes(key.size_bytes) : '—'}</td>
	{/snippet}
</DataTable>

<style>
	.mono {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.empty {
		color: var(--color-text-disabled);
		font-size: var(--text-sm);
	}

	.number {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		text-align: end;
	}
</style>
