<script lang="ts">
	import type { LoginStep, LoginStepSpec } from '#lib/api/index.js';
	import { Note } from '#lib/components/ui/index.js';
	import { describe, labelFor, markFor } from '../steps';

	type Props = {
		kinds: LoginStepSpec[];
		steps: LoginStep[];
		editable: boolean;
		waiting: boolean;
		onAdd: (step: LoginStep) => void;
	};

	let { kinds, steps, editable, waiting, onAdd }: Props = $props();

	const groups = $derived([
		{ label: 'Run today', kinds: kinds.filter((kind) => kind.implemented && !kind.fixed) },
		{ label: 'Planned', kinds: kinds.filter((kind) => !kind.implemented) }
	]);
</script>

<aside class="palette" aria-label="Steps">
	<h2>Steps</h2>
	<Note>
		{waiting
			? 'Choose the step to add where you clicked +.'
			: 'Drag a step onto the flow, or click it to add it at the end.'}
	</Note>

	{#each groups as group (group.label)}
		<h3>{group.label}</h3>
		<ul>
			{#each group.kinds as kind (kind.step)}
				{@const mark = markFor(kind.step)}
				{@const used = steps.includes(kind.step)}
				<li>
					<button
						type="button"
						class="item"
						class:planned={!kind.implemented}
						class:waiting
						disabled={!editable || used}
						draggable={editable && !used}
						title={used ? 'Already in this flow' : describe(kind.step, kinds)}
						ondragstart={(event) => {
							event.dataTransfer?.setData('application/x-login-step', kind.step);
							if (event.dataTransfer) event.dataTransfer.effectAllowed = 'copy';
						}}
						onclick={() => onAdd(kind.step)}
					>
						<span class="mark"><i class={mark} aria-hidden="true"></i></span>
						<span class="text">
							<span class="label">{labelFor(kind.step, kinds)}</span>
							<span class="description">{describe(kind.step, kinds)}</span>
						</span>
						{#if editable && !used}
							<span class="grip"><i class="ri-draggable" aria-hidden="true"></i></span>
						{/if}
					</button>
				</li>
			{/each}
		</ul>
	{/each}
</aside>

<style>
	.palette {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		overflow-y: auto;
		padding: var(--space-4);
		border-right: 1px solid var(--color-border);
		background: var(--color-surface);
	}

	h2 {
		margin: 0;
		font-size: var(--text-base);
	}

	h3 {
		margin: var(--space-3) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}

	ul {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.item {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		background: var(--color-surface);
		color: var(--color-text);
		font: inherit;
		text-align: left;
		cursor: grab;
		transition:
			border-color var(--speed-fast),
			background-color var(--speed-fast);
	}

	.item:hover:not(:disabled),
	.item.waiting:not(:disabled) {
		border-color: var(--color-info);
		background: var(--surface-info);
	}

	.item:disabled {
		cursor: default;
		opacity: 0.5;
	}

	.item.planned {
		border-style: dashed;
	}

	.mark {
		display: inline-flex;
		flex: none;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
	}

	.text {
		display: flex;
		flex: 1;
		flex-direction: column;
		min-width: 0;
	}

	.label {
		font-weight: 600;
		font-size: var(--text-sm);
	}

	.description {
		color: var(--color-text-hint);
		font-size: 12px;
		line-height: 1.35;
	}

	.grip {
		display: inline-flex;
		color: var(--color-text-disabled);
	}
</style>
