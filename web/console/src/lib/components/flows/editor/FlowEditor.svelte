<script lang="ts">
	import { beforeNavigate, goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { SvelteFlowProvider } from '@xyflow/svelte';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { flowsApi, type LoginFlow, type LoginStep, type LoginStepSpec } from '#lib/api/index.js';
	import {
		Alert,
		Button,
		ConfirmDialog,
		IconButton,
		PageHeader,
		Tag,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { saveJson } from '#lib/utils/download.js';
	import { draftOf, freeSlug, problemsOf, toFile, type FlowDraft } from '../steps';
	import FlowCanvas from './FlowCanvas.svelte';
	import Inspector from './Inspector.svelte';
	import Palette from './Palette.svelte';

	type Props = {
		flow: LoginFlow | null;
		initial: FlowDraft;
		kinds: LoginStepSpec[];
		taken: string[];
		editable: boolean;
	};

	let { flow, initial, kinds, taken, editable }: Props = $props();

	const queryClient = useQueryClient();

	// svelte-ignore state_referenced_locally
	let draft = $state(draftOf(initial));
	// svelte-ignore state_referenced_locally
	let saved = $state(JSON.stringify(draftOf(initial)));
	const creating = $derived(flow === null);
	const dirty = $derived(creating || JSON.stringify(draft) !== saved);
	const problems = $derived(problemsOf(draft));

	let selected = $state('start');
	let waiting = $state<number | null>(null);

	let busy = $state(false);
	let confirmingDelete = $state(false);
	let leaving = false;

	let leavingFor = $state<URL | null>(null);
	let confirmingLeave = $state(false);

	beforeNavigate(({ cancel, to, type, shallow }) => {
		if (shallow && type === 'goto') return;
		if (leaving || !editable || !dirty || creating) return;

		cancel();

		// Closing the tab or reloading it can only be asked about by the
		// browser, in its own words: cancelling is what makes it ask.
		if (type === 'leave' || !to) return;

		leavingFor = to.url;
		confirmingLeave = true;
	});

	async function leaveAnyway() {
		if (!leavingFor) return;

		leaving = true;
		confirmingLeave = false;
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(leavingFor);
	}

	function insert(index: number, step: LoginStep) {
		if (draft.steps.includes(step)) return;

		const at = Math.max(1, Math.min(index, draft.steps.length));
		draft.steps = [...draft.steps.slice(0, at), step, ...draft.steps.slice(at)];
		waiting = null;
		selected = step;
	}

	function add(step: LoginStep) {
		insert(waiting ?? draft.steps.length, step);
	}

	function remove(step: LoginStep) {
		draft.steps = draft.steps.filter((one) => one !== step);
		// The requirement lives on the step that sets a factor up and asks
		// for it, so it goes with it.
		if (step === 'totp') draft.require_second_factor = false;
		if (selected === step) selected = 'start';
		waiting = null;
	}

	function lifetimeText(hours: number): string {
		return hours % 24 === 0 && hours >= 24 ? `${hours / 24} d` : `${hours} h`;
	}

	function summaryOf(step: LoginStep): string[] {
		switch (step) {
			case 'identifier':
				return [
					draft.allow_registration ? 'sign-ups' : 'no sign-ups',
					...(draft.require_verified_email ? ['verified address'] : [])
				];
			case 'password':
				return [draft.allow_password_reset ? 'reset link' : 'no reset link'];
			case 'totp':
				return [draft.require_second_factor ? 'required' : 'optional'];
			default:
				return [];
		}
	}

	async function run(action: () => Promise<void>) {
		busy = true;
		try {
			await action();
		} catch (err) {
			notify.error(err, 'Could not save this flow');
		} finally {
			busy = false;
		}
	}

	function body(of: FlowDraft) {
		// The identifier is in every address the flow is known by, so it is
		// only sent when the flow is made.
		const { slug, ...rest } = draftOf(of);
		return creating ? { ...rest, slug } : rest;
	}

	const save = () =>
		run(async () => {
			const result = creating
				? await flowsApi.create(body(draft))
				: await flowsApi.update(flow!.id, body(draft));

			await queryClient.invalidateQueries({ queryKey: keys.flows.all });
			saved = JSON.stringify(draftOf(draft));
			notify.success(creating ? `${result.flow.name} created` : 'Flow saved');

			if (creating) {
				leaving = true;
				await goto(resolve('/(panel)/flows/[id]', { id: result.flow.id }));
			}
		});

	const duplicate = () =>
		run(async () => {
			const copy: FlowDraft = {
				...draftOf(draft),
				name: `Copy of ${draft.name}`,
				slug: freeSlug(`${draft.slug}-copy`, taken),
				is_default: false
			};
			const result = await flowsApi.create(copy);
			await queryClient.invalidateQueries({ queryKey: keys.flows.all });
			notify.success(`${result.flow.name} created`);

			leaving = true;
			await goto(resolve('/(panel)/flows/[id]', { id: result.flow.id }));
		});

	const remove_ = () =>
		run(async () => {
			await flowsApi.remove(flow!.id);
			await queryClient.invalidateQueries({ queryKey: keys.flows.all });
			notify.success(`${flow!.name} deleted`);

			leaving = true;
			await goto(resolve('/(panel)/flows'));
		});

	function discard() {
		draft = JSON.parse(saved);
		selected = 'start';
		waiting = null;
	}
</script>

<SvelteFlowProvider>
	<PageHeader
		crumbs={[
			'Dashboard',
			{ label: 'Login flows', href: resolve('/(panel)/flows') },
			draft.name || 'Untitled flow'
		]}
	>
		{#snippet meta()}
			{#if flow?.is_default}<Tag tone="info" strong>default</Tag>{/if}
			{#if !draft.is_enabled}<Tag>off</Tag>{/if}
			{#if editable && dirty && !creating}<Tag tone="warning">unsaved</Tag>{/if}
		{/snippet}

		{#snippet secondary()}
			<IconButton
				icon="ri-download-2-line"
				label="Export as JSON"
				onclick={() => saveJson(`${draft.slug || 'flow'}.flow.json`, toFile(draft))}
			/>
			{#if editable && !creating}
				<IconButton
					icon="ri-file-copy-line"
					label="Duplicate"
					onclick={duplicate}
					disabled={busy}
				/>
			{/if}
		{/snippet}

		{#snippet actions()}
			{#if editable}
				{#if !creating && !flow?.is_default}
					<Button variant="subtle" colorPalette="danger" onclick={() => (confirmingDelete = true)}
						><i class="ri-delete-bin-line" aria-hidden="true"></i>Delete</Button
					>
				{/if}
				{#if dirty && !creating}
					<Button variant="subtle" onclick={discard} disabled={busy}>
						<i class="ri-arrow-go-back-line" aria-hidden="true"></i>
						Discard changes
					</Button>
				{/if}
				<Button onclick={save} loading={busy} disabled={!dirty || problems.length > 0}>
					<i class="ri-check-line" aria-hidden="true"></i>
					{creating ? 'Create flow' : 'Save changes'}
				</Button>
			{/if}
		{/snippet}
	</PageHeader>

	{#if problems.length > 0 || !editable}
		<div class="messages">
			{#if problems.length > 0}
				<Alert tone="warning">
					<span class="problems">
						{#each problems as problem (problem)}
							<span>{problem}</span>
						{/each}
					</span>
				</Alert>
			{/if}
			{#if !editable}<Alert tone="info"
					>Your roles let you look at login flows, not change them.</Alert
				>{/if}
		</div>
	{/if}

	<div class="workspace">
		<Palette {kinds} steps={draft.steps} {editable} waiting={waiting !== null} onAdd={add} />

		<FlowCanvas
			steps={draft.steps}
			{kinds}
			{summaryOf}
			startDetail={draft.name || 'Untitled flow'}
			endDetail={`session: ${lifetimeText(draft.session_lifetime_hours)}`}
			{selected}
			{editable}
			{waiting}
			onSelect={(id) => (selected = id)}
			onReorder={(steps) => (draft.steps = steps)}
			onInsert={insert}
			onRemove={remove}
			onWait={(index) => (waiting = index)}
		/>

		<Inspector
			bind:draft
			{kinds}
			{selected}
			{editable}
			{creating}
			savedDefault={flow?.is_default ?? false}
			applications={flow?.applications ?? 0}
			onRemove={remove}
		/>
	</div>
</SvelteFlowProvider>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Delete ${draft.name || 'this flow'}?`}
	description={`${flow?.applications ?? 0} ${(flow?.applications ?? 0) === 1 ? 'application falls' : 'applications fall'} back to the default flow. This cannot be undone.`}
	confirmLabel="Delete flow"
	{busy}
	onConfirm={async () => {
		await remove_();
		confirmingDelete = false;
	}}
/>

<ConfirmDialog
	bind:open={confirmingLeave}
	title="Leave without saving?"
	description="This flow has changes that are not saved. Leaving throws them away."
	tone="warning"
	confirmLabel="Leave anyway"
	cancelLabel="Keep editing"
	onConfirm={leaveAnyway}
/>

<style>
	.messages {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.problems {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.workspace {
		display: grid;
		grid-template-columns: 260px minmax(0, 1fr) 320px;
		overflow: hidden;
		height: calc(100dvh - var(--header-height) - 180px);
		min-height: 560px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		background: var(--color-surface);
	}

	@media (max-width: 64rem) {
		.workspace {
			grid-template-columns: minmax(0, 1fr);
			grid-template-rows: auto 520px auto;
			height: auto;
		}

		.workspace :global(.palette),
		.workspace :global(.inspector) {
			border: none;
			border-bottom: 1px solid var(--color-border);
		}
	}
</style>
