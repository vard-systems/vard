<script lang="ts" module>
	import type { Edge } from '@xyflow/svelte';

	export type InsertEdgeData = {
		index: number;
		onInsert?: (index: number) => void;
		waiting: boolean;
		label: string;
	};

	export type InsertEdgeType = Edge<InsertEdgeData, 'insert'>;
</script>

<script lang="ts">
	import { BaseEdge, EdgeLabel, getStraightPath, type EdgeProps } from '@xyflow/svelte';
	import { RiAddLine } from 'svelte-remixicon';
	import { Icon } from '#lib/components/ui/index.js';

	let { id, sourceX, sourceY, targetX, targetY, data }: EdgeProps<InsertEdgeType> = $props();

	const path = $derived(getStraightPath({ sourceX, sourceY, targetX, targetY }));
</script>

<BaseEdge {id} path={path[0]} class="flow-edge" />

{#if data?.onInsert}
	<EdgeLabel x={path[1]} y={path[2]}>
		<button
			type="button"
			class="insert nodrag nopan"
			class:waiting={data.waiting}
			title={data.label}
			aria-label={data.label}
			onclick={() => data.onInsert?.(data.index)}
		>
			<Icon icon={RiAddLine} size="0.9rem" />
		</button>
	</EdgeLabel>
{/if}

<style>
	.insert {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 22px;
		height: 22px;
		padding: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-surface);
		color: var(--color-text-hint);
		cursor: pointer;
		transition:
			background-color var(--speed-fast),
			color var(--speed-fast),
			transform var(--speed-fast);
	}

	.insert:hover,
	.insert.waiting {
		border-color: var(--color-info);
		background: var(--color-info);
		color: var(--color-surface);
		transform: scale(1.1);
	}
</style>
