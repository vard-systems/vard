<script lang="ts">
	import '@xyflow/svelte/dist/style.css';
	import {
		Background,
		Controls,
		SvelteFlow,
		useSvelteFlow,
		type Edge,
		type Node
	} from '@xyflow/svelte';
	import type { LoginStep, LoginStepSpec } from '#lib/api/index.js';
	import { describe, labelFor, markFor, specFor } from '../steps';
	import EndNode, { type EndNodeData } from './EndNode.svelte';
	import InsertEdge, { type InsertEdgeData } from './InsertEdge.svelte';
	import StepNode, { type StepNodeData } from './StepNode.svelte';

	type Props = {
		steps: LoginStep[];
		kinds: LoginStepSpec[];
		summaryOf: (step: LoginStep) => string[];
		startDetail: string;
		endDetail: string;
		selected: string;
		editable: boolean;
		waiting: number | null;
		onSelect: (id: string) => void;
		onReorder: (steps: LoginStep[]) => void;
		onInsert: (index: number, step: LoginStep) => void;
		onRemove: (step: LoginStep) => void;
		onWait: (index: number | null) => void;
	};

	let {
		steps,
		kinds,
		summaryOf,
		startDetail,
		endDetail,
		selected,
		editable,
		waiting,
		onSelect,
		onReorder,
		onInsert,
		onRemove,
		onWait
	}: Props = $props();

	const { screenToFlowPosition, fitView } = useSvelteFlow();

	const ROW = 150;
	const nodeTypes = { step: StepNode, end: EndNode };
	const edgeTypes = { insert: InsertEdge };

	/** Bumped when a drag ends somewhere that changes nothing, so the nodes
	    are laid out again rather than left where they were dropped. */
	let revision = $state(0);

	/** The flow laid out from its steps. Svelte Flow writes a node's position
	    while it is dragged — a derived value can be overwritten for a while —
	    and the next change to the steps lays it out afresh: the steps are the
	    truth, and every change comes back through them. */
	let nodes = $derived.by((): Node[] => {
		void revision;
		const drawn: Node[] = [
			{
				id: 'start',
				type: 'end',
				position: { x: 0, y: 0 },
				draggable: false,
				selected: selected === 'start',
				data: {
					kind: 'start',
					label: 'Sign-in starts',
					detail: startDetail,
					icon: 'ri-login-circle-line'
				} satisfies EndNodeData
			}
		];

		steps.forEach((step, index) => {
			const spec = specFor(step, kinds);
			const fixed = spec?.fixed ?? index === 0;

			drawn.push({
				id: step,
				type: 'step',
				position: { x: 0, y: 90 + index * ROW },
				draggable: editable && !fixed,
				selected: selected === step,
				data: {
					step,
					label: labelFor(step, kinds),
					description: describe(step, kinds),
					icon: markFor(step),
					fixed,
					planned: spec?.implemented === false,
					summary: summaryOf(step),
					onRemove: editable && !fixed ? () => onRemove(step) : undefined,
					removeLabel: 'Remove step',
					plannedLabel: 'Not run yet',
					fixedLabel: 'Always first'
				} satisfies StepNodeData
			});
		});

		drawn.push({
			id: 'end',
			type: 'end',
			position: { x: 0, y: 90 + steps.length * ROW + 10 },
			draggable: false,
			selected: selected === 'end',
			data: {
				kind: 'end',
				label: 'Signed in',
				detail: endDetail,
				icon: 'ri-flag-line'
			} satisfies EndNodeData
		});

		return drawn;
	});

	let edges = $derived.by((): Edge[] =>
		nodes.slice(1).map((node, index) => ({
			id: `${nodes[index].id}->${node.id}`,
			source: nodes[index].id,
			target: node.id,
			type: 'insert',
			selectable: false,
			data: {
				index,
				onInsert:
					editable && index > 0 ? (at: number) => onWait(waiting === at ? null : at) : undefined,
				waiting: waiting === index,
				label: 'Add a step here'
			} satisfies InsertEdgeData
		}))
	);

	function gapAt(y: number): number {
		let index = 1;
		steps.forEach((_, at) => {
			if (at > 0 && 90 + at * ROW + ROW / 2 < y) index = at + 1;
		});
		return Math.min(index, steps.length);
	}

	function reorder() {
		const moved = nodes
			.filter((node) => node.type === 'step' && node.id !== steps[0])
			.sort((a, b) => a.position.y - b.position.y)
			.map((node) => node.id as LoginStep);

		const next = [steps[0], ...moved];
		if (next.join() === steps.join()) revision++;
		else onReorder(next);
	}

	function drop(event: DragEvent) {
		event.preventDefault();
		const step = event.dataTransfer?.getData('application/x-login-step') as LoginStep | undefined;
		if (!step || !editable) return;

		const point = screenToFlowPosition({ x: event.clientX, y: event.clientY });
		onInsert(gapAt(point.y), step);
	}

	function keydown(event: KeyboardEvent) {
		if (!editable || (event.key !== 'Delete' && event.key !== 'Backspace')) return;

		const step = steps.find((one) => one === selected);
		const spec = step && specFor(step, kinds);
		if (step && step !== steps[0] && !spec?.fixed) {
			event.preventDefault();
			onRemove(step);
		}
	}

	let fitted = false;

	let count = 0;
	$effect(() => {
		const now = steps.length;
		if (count !== 0 && now !== count) {
			setTimeout(() => void fitView({ padding: 0.2, maxZoom: 1, duration: 250 }), 30);
		}
		count = now;
	});
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="canvas"
	ondragover={(event) => {
		if (editable) event.preventDefault();
	}}
	ondrop={drop}
	onkeydown={keydown}
>
	<SvelteFlow
		bind:nodes
		bind:edges
		{nodeTypes}
		{edgeTypes}
		nodesConnectable={false}
		deleteKey={null}
		elementsSelectable
		minZoom={0.3}
		maxZoom={1.5}
		fitView
		fitViewOptions={{ padding: 0.2, maxZoom: 1 }}
		oninit={() => {
			if (!fitted) void fitView({ padding: 0.2, maxZoom: 1 });
			fitted = true;
		}}
		onnodeclick={({ node }) => onSelect(node.id)}
		onpaneclick={() => onSelect('start')}
		onnodedragstop={reorder}
		proOptions={{ hideAttribution: true }}
	>
		<Background gap={20} />
		<Controls showLock={false} position="bottom-right" />
	</SvelteFlow>
</div>

<style>
	.canvas {
		position: relative;
		height: 100%;
		min-height: 520px;
		background: var(--color-body);
	}

	.canvas :global(.svelte-flow) {
		--xy-background-color: var(--color-body);
		--xy-background-pattern-color: var(--color-border);
		--xy-edge-stroke: var(--color-text-disabled);
		--xy-edge-stroke-width: 1.5;
		--xy-controls-button-background-color: var(--color-surface);
		--xy-controls-button-background-color-hover: var(--color-secondary);
		--xy-controls-button-color: var(--color-text);
		--xy-controls-button-border-color: var(--color-border);
		--xy-node-border-radius: var(--radius-surface);
		--xy-edge-label-background-color: transparent;
		--xy-edge-label-color: inherit;
		--xy-edge-stroke-selected: var(--color-info);
		--xy-connectionline-stroke: var(--color-info);
		--xy-handle-background-color: var(--color-text-disabled);
		--xy-handle-border-color: var(--color-body);
		--xy-selection-background-color: var(--surface-info);
		--xy-selection-border: 1px dotted var(--color-info);
		--xy-minimap-background-color: var(--color-surface);
		--xy-attribution-background-color: transparent;
		background: var(--color-body);
	}

	.canvas :global(.svelte-flow__node) {
		padding: 0;
		border: none;
		background: transparent;
		box-shadow: none;
	}

	.canvas :global(.svelte-flow__controls) {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-sm);
		box-shadow: none;
	}
</style>
