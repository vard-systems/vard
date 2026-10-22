<script lang="ts">
	import type { FieldType } from '#lib/api/index.js';
	import { Tag } from '#lib/components/ui/index.js';
	import { formatDate } from '#lib/utils/format.js';

	type Props = {
		type: FieldType;
		value: unknown;
	};

	let { type, value }: Props = $props();

	const empty = $derived(value === null || value === undefined || value === '');
</script>

{#if empty}
	<span class="empty">N/A</span>
{:else if type === 'bool'}
	<Tag tone={value ? 'success' : 'neutral'} dot>{value ? 'Yes' : 'No'}</Tag>
{:else if type === 'number'}
	<span class="mono">{Number(value).toLocaleString()}</span>
{:else if type === 'date'}
	<span class="mono">{formatDate(String(value))}</span>
{:else}
	<span class="text">{value}</span>
{/if}

<style>
	.empty {
		color: var(--color-text-disabled);
		font-size: var(--text-sm);
	}

	.mono {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.text {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
</style>
