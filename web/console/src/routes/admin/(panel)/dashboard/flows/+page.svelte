<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { RiAddLine, RiCloseLine, RiRefreshLine, RiUpload2Line } from 'svelte-remixicon';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { flowsApi, type LoginFlow } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		Button,
		Icon,
		IconButton,
		PageHeader,
		SearchInput,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import FlowTable from '#lib/components/flows/FlowTable.svelte';
	import { TEMPLATES, freeSlug, fromFile } from '#lib/components/flows/steps.js';
	import { can } from '#lib/permissions.js';
	import { firstPage, keys, LIST_PAGE_SIZE, loginFlowsOptions } from '#lib/query/index.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const flows = createQuery(() =>
		loginFlowsOptions({ flows: data.flows, step_kinds: data.stepKinds })
	);

	const canWrite = $derived(can(data.admin, 'login_flows.write'));

	let search = $derived(data.search);

	const visible = $derived.by(() => {
		const term = data.search.trim().toLowerCase();
		if (term === '') return flows.data.flows;

		return flows.data.flows.filter((flow) =>
			[flow.name, flow.slug, flow.description].some((value) => value.toLowerCase().includes(term))
		);
	});

	let shown = $derived(firstPage(data.search));
	const rows = $derived(visible.slice(0, shown));

	let choosing = $state(false);
	let importing = $state(false);
	let file = $state<HTMLInputElement>();

	async function apply(changes: { search?: string }) {
		const params = new SvelteURLSearchParams(page.url.search);

		for (const [key, value] of Object.entries(changes)) {
			if (value) params.set(key, value);
			else params.delete(key);
		}

		const query = params.toString();
		const path = resolve('/admin/(panel)/dashboard/flows');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(query ? `${path}?${query}` : path, { reset: false });
	}

	let timer: ReturnType<typeof setTimeout>;

	function debounced() {
		clearTimeout(timer);
		timer = setTimeout(() => apply({ search }), 250);
	}

	function open(flow: LoginFlow) {
		goto(resolve('/admin/(panel)/dashboard/flows/[id]', { id: flow.id }));
	}

	async function importFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const chosen = input.files?.[0];
		input.value = '';
		if (!chosen) return;

		importing = true;

		try {
			let draft;
			try {
				draft = fromFile(await chosen.text(), flows.data.step_kinds);
			} catch (err) {
				notify.problem((err as Error).message);
				return;
			}

			const taken = flows.data.flows.map((flow) => flow.slug);
			const result = await flowsApi.create({
				...draft,
				slug: freeSlug(draft.slug, taken),
				is_default: false
			});
			await queryClient.invalidateQueries({ queryKey: keys.flows.all });
			notify.success(`${result.flow.name} imported`);
			await goto(resolve('/admin/(panel)/dashboard/flows/[id]', { id: result.flow.id }));
		} catch (err) {
			notify.error(err, 'Could not import this flow');
		} finally {
			importing = false;
		}
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.flows.all }),
				new Promise((done) => setTimeout(done, 400))
			]);
		} finally {
			refreshing = false;
		}
	}
</script>

<svelte:head><title>Login flows · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Login flows']}
	count={flows.data.flows.length}
	description="How people sign in, drawn as the steps they go through. Every application follows its own flow or the default one: passwords, other accounts, verified addresses and how long a session lasts are all decided here."
>
	{#snippet secondary()}
		<IconButton
			icon={RiRefreshLine}
			label="Refresh the data"
			onclick={refresh}
			loading={refreshing}
			disabled={refreshing}
		/>
	{/snippet}

	{#snippet actions()}
		{#if canWrite}
			<input
				id="flow-import-file"
				name="flow-import-file"
				bind:this={file}
				type="file"
				accept="application/json,.json"
				hidden
				onchange={importFile}
			/>
			<Button variant="subtle" loading={importing} onclick={() => file?.click()}>
				<Icon icon={RiUpload2Line} />
				Import
			</Button>
			<Button onclick={() => (choosing = !choosing)}>
				<Icon icon={choosing ? RiCloseLine : RiAddLine} />
				New flow
			</Button>
		{/if}
	{/snippet}
</PageHeader>

{#if choosing}
	<section class="templates" aria-label="Start from a template">
		<h2>Start from a template</h2>
		<div class="cards">
			{#each TEMPLATES as template (template.id)}
				<!-- The path is resolved; only the query is added to it. -->
				<!-- eslint-disable svelte/no-navigation-without-resolve -->
				<a
					class="card"
					href={`${resolve('/admin/(panel)/dashboard/flows/new')}?template=${template.id}`}
				>
					<span class="mark"><Icon icon={template.icon} size="1.2rem" /></span>
					<strong>{template.name}</strong>
					<span>{template.hint}</span>
				</a>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
			{/each}
		</div>
	</section>
{/if}

<Toolbar>
	<SearchInput
		label="Search login flows"
		placeholder="Search name, identifier or description…"
		bind:value={search}
		onsubmit={() => apply({ search })}
		oninput={debounced}
	/>
</Toolbar>

<FlowTable
	flows={rows}
	kinds={flows.data.step_kinds}
	onOpen={open}
	empty={flows.data.flows.length === 0 ? 'No flows yet.' : 'No flows match this.'}
/>

{#if visible.length > shown}
	<ShowMore {shown} total={visible.length} onclick={() => (shown += LIST_PAGE_SIZE)} />
{/if}

<style>
	.templates h2 {
		margin: 0 0 var(--space-2);
		font-size: var(--text-base);
	}

	.cards {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: var(--space-3);
	}

	.card {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
		padding: var(--space-4);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		background: var(--color-surface);
		color: var(--color-text);
		text-decoration: none;
		transition:
			border-color var(--speed-fast),
			background-color var(--speed-fast);
	}

	.card:hover {
		border-color: var(--color-info);
		background: var(--surface-info);
	}

	.card span:last-child {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.mark {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		margin-bottom: var(--space-1);
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
	}
</style>
