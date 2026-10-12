<script lang="ts">
	import type { FieldType } from '$lib/api';
	import { Badge } from '$lib/components/ui';
	import { formatDate } from '$lib/utils/format';

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
	<Badge tone={value ? 'success' : 'neutral'}>{value ? 'True' : 'False'}</Badge>
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
