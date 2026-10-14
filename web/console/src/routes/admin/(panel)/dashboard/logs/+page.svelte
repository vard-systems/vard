<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { createInfiniteQuery, useQueryClient } from '@tanstack/svelte-query';
	import { RiDownload2Line, RiFilterOffLine, RiRefreshLine } from 'svelte-remixicon';
	import { activityApi } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import ActivityTable from '#lib/components/activity/ActivityTable.svelte';
	import { actionsIn } from '#lib/components/activity/actions.js';
	import { kindLabels, type Kind } from '#lib/components/activity/filters.js';
	import {
		Button,
		DatePicker,
		FilterChip,
		Icon,
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

	async function apply(changes: Partial<Record<'q' | 'kind' | 'actor' | 'from' | 'to', string>>) {
		const params = new SvelteURLSearchParams(page.url.search);

		for (const [key, value] of Object.entries(changes)) {
			if (value) params.set(key, value);
			else params.delete(key);
		}

		const query = params.toString();
		const path = resolve('/admin/(panel)/dashboard/logs');

		// resolve() has already applied any base path; the query is only ever
		// appended to what it returned.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(query ? `${path}?${query}` : path, { reset: false });
	}

	function clear() {
		search = '';
		void apply({ q: '', kind: '', actor: '', from: '', to: '' });
	}

	let timer: ReturnType<typeof setTimeout>;

	function debounced() {
		clearTimeout(timer);
		timer = setTimeout(() => apply({ q: search.trim() }), 300);
	}

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
	<title>Logs · {BRAND.name}</title>
</svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Logs']}
	description="Every sign-in and every change administrators made, newest first. Filters apply to the whole log, not just the page shown."
>
	{#snippet secondary()}
		<IconButton
			icon={RiRefreshLine}
			label="Refresh"
			onclick={refresh}
			loading={refreshing}
			disabled={refreshing}
		/>
	{/snippet}

	{#snippet actions()}
		<LinkButton href={activityApi.exportUrl(data.view.filter)} variant="subtle" download>
			<Icon icon={RiDownload2Line} />
			Export CSV
		</LinkButton>
	{/snippet}
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search the log"
		placeholder="Search who, what, where or an id…"
		bind:value={search}
		onsubmit={() => apply({ q: search.trim() })}
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
			onChange={(kind) => apply({ kind })}
		/>
	</div>

	<div class="dates">
		<DatePicker
			compact
			clearable
			label="From"
			value={data.view.from}
			max={data.view.to || today}
			onChange={(from) => apply({ from })}
		/>
		<DatePicker
			compact
			clearable
			label="To"
			value={data.view.to}
			min={data.view.from || undefined}
			max={today}
			onChange={(to) => apply({ to })}
		/>
	</div>

	{#if data.view.actor}
		<FilterChip title="Show everyone" onclear={() => apply({ actor: '' })}>
			by <strong>{data.view.actor}</strong>
		</FilterChip>
	{/if}

	{#snippet end()}
		{#if filtered}
			<Button variant="ghost" onclick={clear}>
				<Icon icon={RiFilterOffLine} />
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
