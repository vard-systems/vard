<script lang="ts" module>
	import type { Node } from '@xyflow/svelte';
	import type { LoginStep } from '#lib/api/index.js';

	export type StepNodeData = {
		step: LoginStep;
		label: string;
		description: string;
		icon: IconData;
		fixed: boolean;
		planned: boolean;
		summary: string[];
		onRemove?: () => void;
		removeLabel: string;
		plannedLabel: string;
		fixedLabel: string;
	};

	export type StepNodeType = Node<StepNodeData, 'step'>;
</script>

<script lang="ts">
	import { Handle, Position, type NodeProps } from '@xyflow/svelte';
	import { RiCloseLine, RiLock2Line, type IconData } from '#lib/icons.js';
	import { Icon } from '#lib/components/ui/index.js';

	let { data, selected }: NodeProps<StepNodeType> = $props();
</script>

<div class="card" class:selected class:planned={data.planned}>
	<Handle type="target" position={Position.Top} isConnectable={false} />

	<span class="mark"><Icon icon={data.icon} size="1.1rem" /></span>

	<div class="body">
		<span class="title">
			{data.label}
			{#if data.fixed}
				<span class="fixed" title={data.fixedLabel}><Icon icon={RiLock2Line} size="0.8rem" /></span>
			{/if}
		</span>
		<span class="description">{data.description}</span>

		{#if data.planned || data.summary.length > 0}
			<span class="chips">
				{#if data.planned}<span class="chip warning">{data.plannedLabel}</span>{/if}
				{#each data.summary as item (item)}
					<span class="chip">{item}</span>
				{/each}
			</span>
		{/if}
	</div>

	{#if data.onRemove}
		<button
			type="button"
			class="remove nodrag"
			title={data.removeLabel}
			aria-label={data.removeLabel}
			onclick={(event) => {
				event.stopPropagation();
				data.onRemove?.();
			}}
		>
			<Icon icon={RiCloseLine} size="0.95rem" />
		</button>
	{/if}

	<Handle type="source" position={Position.Bottom} isConnectable={false} />
</div>

<style>
	.card {
		position: relative;
		display: flex;
		gap: var(--space-3);
		width: 300px;
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text);
		box-shadow: var(--shadow-sm, 0 1px 2px rgb(0 0 0 / 6%));
		transition:
			border-color var(--speed-fast),
			box-shadow var(--speed-fast);
	}

	.card:hover {
		border-color: var(--color-text-hint);
	}

	.card.selected {
		border-color: var(--color-info);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-info) 20%, transparent);
	}

	.card.planned {
		border-style: dashed;
	}

	.mark {
		display: inline-flex;
		flex: none;
		align-items: center;
		justify-content: center;
		width: 34px;
		height: 34px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.title {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		font-weight: 600;
	}

	.fixed {
		display: inline-flex;
		color: var(--color-text-hint);
	}

	.description {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.35;
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-top: 4px;
	}

	.chip {
		padding: 1px 7px;
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		font-size: 11px;
		line-height: 18px;
	}

	.chip.warning {
		background: var(--surface-warning);
		color: color-mix(in srgb, var(--color-warning) 70%, var(--color-text));
	}

	.remove {
		position: absolute;
		top: 6px;
		right: 6px;
		display: inline-flex;
		padding: 3px;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text-hint);
		cursor: pointer;
		opacity: 0;
		transition: opacity var(--speed-fast);
	}

	.card:hover .remove,
	.card.selected .remove,
	.remove:focus-visible {
		opacity: 1;
	}

	.remove:hover {
		background: var(--color-secondary);
		color: var(--color-text);
	}

	.card :global(.svelte-flow__handle) {
		opacity: 0;
	}
</style>
