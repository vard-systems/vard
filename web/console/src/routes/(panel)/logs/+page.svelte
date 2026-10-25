<script lang="ts">
	import { createInfiniteQuery, useQueryClient } from '@tanstack/svelte-query';
	import { activityApi } from '#lib/api/index.js';
	import { pageTitle } from '#lib/brand.js';
	import ActivityTable from '#lib/components/activity/ActivityTable.svelte';
	import { actionsIn } from '#lib/components/activity/actions.js';
	import { kindLabels, type Kind } from '#lib/components/activity/filters.js';
	import {
		Button,
		DatePicker,
		FilterChip,
		IconButton,
		LinkButton,
		PageHeader,
		SearchInput,
		Select,
		ShowMore,
		Toolbar,
		todayIso
	} from '#lib/components/ui/index.js';
	import { keys, logsOptions } from '#lib/query/index.js';
	import type { PageData } from './$types';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const logs = createInfiniteQuery(() => logsOptions(data.view.filter, data.first));

	const entries = $derived(logs.data?.pages.flatMap((one) => one.logs) ?? []);

	// What is typed in the search box, applied once the typing pauses; the
	// address is what the list follows.
	// svelte-ignore state_referenced_locally
	let search = $state(data.view.q);

	const kinds = (Object.keys(kindLabels) as Kind[])
		.filter((kind) => kind === 'refused' || actionsIn(kind).length > 0)
		.map((kind) => ({ value: kind, label: kindLabels[kind] }));

	const filtered = $derived(
		Boolean(data.view.q || data.view.kind || data.view.actor || data.view.from || data.view.to)
	);

	function clear() {
		search = '';
		void setSearchParams({ q: '', kind: '', actor: '', from: '', to: '' });
	}

	const debounced = debounce(() => setSearchParams({ q: search.trim() }), 300);

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;
		try {
			await Promise.all([
				queryClient.resetQueries({ queryKey: keys.logs.all }),
				new Promise((done) => setTimeout(done, 400))
			]);
		} finally {
			refreshing = false;
		}
	}

	const today = todayIso();
</script>

<svelte:head>
	<title>{pageTitle('Logs')}</title>
</svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Logs']}
	description="Every sign-in and every change administrators made, newest first. Filters apply to the whole log, not just the page shown."
>
	{#snippet secondary()}
		<IconButton
			icon="ri-refresh-line"
			label="Refresh"
			onclick={refresh}
			loading={refreshing}
			disabled={refreshing}
		/>
	{/snippet}

	{#snippet actions()}
		<LinkButton href={activityApi.exportUrl(data.view.filter)} variant="subtle" download>
			<i class="ri-download-2-line" aria-hidden="true"></i>
			Export CSV
		</LinkButton>
	{/snippet}
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search the log"
		placeholder="Search who, what, where or an id…"
		bind:value={search}
		onsubmit={() => setSearchParams({ q: search.trim() })}
		oninput={debounced}
	/>

	<div class="kind">
		<Select
			compact
			label="Kind"
			value={data.view.kind}
			options={kinds}
			placeholder="Every kind"
			clearable
			onChange={(kind) => setSearchParams({ kind })}
		/>
	</div>

	<div class="dates">
		<DatePicker
			compact
			clearable
			label="From"
			value={data.view.from}
			max={data.view.to || today}
			onChange={(from) => setSearchParams({ from })}
		/>
		<DatePicker
			compact
			clearable
			label="To"
			value={data.view.to}
			min={data.view.from || undefined}
			max={today}
			onChange={(to) => setSearchParams({ to })}
		/>
	</div>

	{#if data.view.actor}
		<FilterChip title="Show everyone" onclear={() => setSearchParams({ actor: '' })}>
			by <strong>{data.view.actor}</strong>
		</FilterChip>
	{/if}

	{#snippet end()}
		{#if filtered}
			<Button variant="ghost" onclick={clear}>
				<i class="ri-filter-off-line" aria-hidden="true"></i>
				Clear filters
			</Button>
		{/if}
	{/snippet}
</Toolbar>

<ActivityTable
	events={entries}
	empty={filtered ? 'No entries match these filters.' : 'Nothing has happened yet.'}
/>

{#if logs.hasNextPage}
	<ShowMore
		shown={entries.length}
		loading={logs.isFetchingNextPage}
		onclick={() => logs.fetchNextPage()}
	/>
{/if}

<style>
	.kind {
		width: 15rem;
	}

	.dates {
		display: flex;
		gap: var(--space-2);
	}

	.dates > :global(*) {
		width: 14rem;
	}

	@media (max-width: 40rem) {
		.kind,
		.dates,
		.dates > :global(*) {
			width: 100%;
		}
	}
</style>
