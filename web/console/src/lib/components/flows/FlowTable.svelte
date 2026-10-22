<script lang="ts">
	import type { LoginFlow, LoginStepSpec } from '#lib/api/index.js';
	import { Badge, type Column, DataTable, Tag, Tooltip } from '#lib/components/ui/index.js';
	import { describe, labelFor, markFor, specFor } from './steps';

	type Props = {
		flows: LoginFlow[];
		kinds: LoginStepSpec[];
		onOpen: (flow: LoginFlow) => void;
		empty: string;
	};

	let { flows, kinds, onOpen, empty }: Props = $props();

	const columns: Column[] = $derived([
		{ key: 'flow', label: 'Flow', min: '14rem' },
		{ key: 'steps', label: 'Steps', min: '18rem', grow: true },
		{ key: 'applications', label: 'Applications', min: '9rem' },
		{ key: 'status', label: 'Status', min: '10rem' }
	]);
</script>

<DataTable {columns} rows={flows} {empty} {onOpen} label={(flow) => `Open ${flow.name}`}>
	{#snippet row(flow)}
		<td>
			<span class="names">
				<strong>{flow.name}</strong>
				<span class="slug">{flow.slug}</span>
			</span>
		</td>

		<td>
			<span class="steps">
				{#each flow.steps as step (step)}
					{@const mark = markFor(step)}
					{@const spec = specFor(step, kinds)}
					<Tooltip
						label={spec?.implemented === false
							? `${describe(step, kinds)} Not run yet.`
							: describe(step, kinds) || step}
					>
						{#snippet children(trigger)}
							<span class="step" class:planned={spec?.implemented === false} {...trigger()}>
								<i class={mark} style:font-size="14px" aria-hidden="true"></i>
								{labelFor(step, kinds)}
							</span>
						{/snippet}
					</Tooltip>
				{/each}
			</span>
		</td>

		<td>
			{#if flow.applications > 0}
				<span class="text">{flow.applications}</span>
			{:else if flow.is_default}
				<span class="empty">everything else</span>
			{:else}
				<span class="empty">none</span>
			{/if}
		</td>

		<td>
			<span class="status">
				{#if flow.is_default}
					<Tag tone="info" strong>default</Tag>
				{:else if flow.is_enabled}
					<Badge tone="success">on</Badge>
				{:else}
					<Badge>off</Badge>
				{/if}

				{#if flow.planned.length > 0}
					<Tag tone="warning" small>{`${flow.planned.length} not run yet`}</Tag>
				{/if}
			</span>
		</td>
	{/snippet}
</DataTable>

<style>
	.names {
		display: flex;
		flex-direction: column;
		min-width: 0;
		line-height: 1.3;
	}

	.slug {
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.steps {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 5px;
	}

	.step {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 7px;
		border: 1px solid var(--color-secondary-alt);
		border-radius: var(--radius-sm);
		background: var(--color-surface-alt);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		white-space: nowrap;
	}

	.step.planned {
		border-style: dashed;
		background: transparent;
	}

	.status {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}

	.empty {
		color: var(--color-text-disabled);
	}
</style>
