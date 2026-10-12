<script lang="ts" module>
	import type { ComponentType } from 'svelte';
	import type { Node } from '@xyflow/svelte';

	export type EndNodeData = {
		kind: 'start' | 'end';
		label: string;
		detail: string;
		icon: ComponentType;
	};

	export type EndNodeType = Node<EndNodeData, 'end'>;
</script>

<script lang="ts">
	import { Handle, Position, type NodeProps } from '@xyflow/svelte';
	import { Icon } from '$lib/components/ui';

	let { data, selected }: NodeProps<EndNodeType> = $props();
</script>

<div class="pill {data.kind}" class:selected>
	{#if data.kind === 'end'}
		<Handle type="target" position={Position.Top} isConnectable={false} />
	{/if}

	<Icon icon={data.icon} size="1rem" />
	<span class="label">{data.label}</span>
	<span class="detail">{data.detail}</span>

	{#if data.kind === 'start'}
		<Handle type="source" position={Position.Bottom} isConnectable={false} />
	{/if}
</div>

<style>
	.pill {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 300px;
		height: 44px;
		padding: 0 var(--space-4);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		color: var(--color-text);
		transition:
			border-color var(--speed-fast),
			box-shadow var(--speed-fast);
	}

	.pill.end {
		background: var(--surface-success);
	}

	.pill.selected {
		border-color: var(--color-info);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-info) 20%, transparent);
	}

	.label {
		font-weight: 600;
	}

	.detail {
		overflow: hidden;
		margin-left: auto;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.pill :global(.svelte-flow__handle) {
		opacity: 0;
	}
</style>
