<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { flowsApi, type LoginFlow } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		Button,
		IconButton,
		PageHeader,
		SearchInput,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import FlowTable from '#lib/components/flows/FlowTable.svelte';
	import TemplatePicker from '#lib/components/flows/TemplatePicker.svelte';
	import { freeSlug, fromFile } from '#lib/components/flows/steps.js';
	import { can } from '#lib/permissions.js';
	import { firstPage, keys, LIST_PAGE_SIZE, loginFlowsOptions } from '#lib/query/index.js';
	import type { PageData } from './$types';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

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

	const debounced = debounce(() => setSearchParams({ search }));

	function open(flow: LoginFlow) {
		void goto(resolve('/(panel)/flows/[id]', { id: flow.id }));
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
			await goto(resolve('/(panel)/flows/[id]', { id: result.flow.id }));
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
			icon="ri-refresh-line"
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
				<i class="ri-upload-2-line" aria-hidden="true"></i>
				Import
			</Button>
			<Button onclick={() => (choosing = !choosing)}>
				{#if choosing}<i class="ri-close-line" aria-hidden="true"></i>{:else}<i
						class="ri-add-line"
						aria-hidden="true"
					></i>{/if}
				New flow
			</Button>
		{/if}
	{/snippet}
</PageHeader>

{#if choosing}
	<TemplatePicker />
{/if}

<Toolbar>
	<SearchInput
		label="Search login flows"
		placeholder="Search name, identifier or description…"
		bind:value={search}
		onsubmit={() => setSearchParams({ search })}
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
