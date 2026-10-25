<script lang="ts">
	import {
		createInfiniteQuery,
		createMutation,
		createQuery,
		useQueryClient
	} from '@tanstack/svelte-query';
	import { cacheApi, type CacheDatabaseName, type CacheKey } from '#lib/api/index.js';
	import { pageTitle } from '#lib/brand.js';
	import { cacheKeysOptions, cacheOverviewOptions, keys } from '#lib/query/index.js';
	import {
		Alert,
		Button,
		ConfirmDialog,
		FilterChip,
		IconButton,
		PageHeader,
		SearchInput,
		SegmentedControl,
		Select,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import CacheGroups from '#lib/components/cache/CacheGroups.svelte';
	import CacheKeyDialog from '#lib/components/cache/CacheKeyDialog.svelte';
	import CacheKeyTable from '#lib/components/cache/CacheKeyTable.svelte';
	import CacheStats from '#lib/components/cache/CacheStats.svelte';
	import { databases, kindFilters } from '#lib/components/cache/cache.js';
	import type { PageData } from './$types';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const overview = createQuery(() => cacheOverviewOptions(data.overview));
	const listing = createInfiniteQuery(() =>
		cacheKeysOptions(
			{ database: data.database, kind: data.kind, group: data.group, search: data.search },
			data.first
		)
	);

	const database = $derived(overview.data.databases.find((one) => one.name === data.database));
	const rows = $derived(listing.data?.pages.flatMap((one) => one.keys) ?? []);

	let search = $derived(data.search);

	const debounced = debounce(() => setSearchParams({ search }));

	function switchTo(name: CacheDatabaseName) {
		void setSearchParams({
			database: name === 'cache' ? '' : name,
			kind: '',
			group: '',
			search: ''
		});
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.cache.all }),
				new Promise((done) => setTimeout(done, 400))
			]);
		} finally {
			refreshing = false;
		}
	}

	let opened = $state<CacheKey | null>(null);
	let dialogOpen = $state(false);

	function open(key: CacheKey) {
		opened = key;
		dialogOpen = true;
	}

	const clear = createMutation(() => ({
		mutationFn: (group: string) => cacheApi.clearGroup(data.database, group),
		onSuccess: () => {
			notify.success('Group cleared');
			return queryClient.invalidateQueries({ queryKey: keys.cache.all });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not clear this group');
		}
	}));

	let flushing = $state(false);

	const flush = createMutation(() => ({
		mutationFn: () => cacheApi.flush(data.database),
		onSuccess: async () => {
			notify.success('Database flushed');
			await queryClient.invalidateQueries({ queryKey: keys.cache.all });
			flushing = false;
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not flush this database');
			flushing = false;
		}
	}));

	const label = $derived(databases[data.database].label);
</script>

<svelte:head><title>{pageTitle('Cache')}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Cache']}
	count={database?.keys}
	description="What this server keeps in Redis, in two databases: a cache of what every page reads, and the sessions and who they belong to. Everything here is a copy the server can read from the database again."
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
		{#if overview.data.configured}
			<Button colorPalette="danger" variant="subtle" onclick={() => (flushing = true)}>
				<i class="ri-delete-bin-2-line" aria-hidden="true"></i>
				Flush {label.toLowerCase()}
			</Button>
		{/if}
	{/snippet}
</PageHeader>

{#if !overview.data.configured}
	<Alert tone="info">
		This server runs without Redis: every page is read from the database, and each server process
		keeps its own rate limits. Set LOGINER_REDIS_HOST to give it one.
	</Alert>
{:else}
	<Toolbar>
		<SegmentedControl
			label="Redis database"
			options={[
				{
					value: 'cache',
					label: `${databases.cache.label} · db ${overview.data.databases[0]?.number ?? 0}`
				},
				{
					value: 'sessions',
					label: `${databases.sessions.label} · db ${overview.data.databases[1]?.number ?? 1}`
				}
			]}
			value={data.database}
			onChange={switchTo}
		/>
		<Select
			label="Kind"
			compact
			placeholder="Every kind"
			value={data.kind}
			options={kindFilters[data.database]}
			onChange={(kind) => setSearchParams({ kind })}
		/>
		<SearchInput
			label="Search keys"
			placeholder="Search key names…"
			bind:value={search}
			onsubmit={() => setSearchParams({ search })}
			oninput={debounced}
		/>
		{#if data.group}
			<FilterChip title="Clear the group filter" onclear={() => setSearchParams({ group: '' })}>
				group: <strong>{data.group}</strong>
			</FilterChip>
		{/if}
	</Toolbar>

	<p class="about">{databases[data.database].description}</p>

	{#if database && !database.available}
		<Alert tone="warning">
			Redis did not answer for this database. The server is reading everything from the database
			until it does.
		</Alert>
	{:else if database}
		<CacheStats {database} />

		{#if database.groups.length > 0}
			<CacheGroups
				groups={database.groups}
				current={data.group}
				clearing={clear.isPending ? clear.variables : undefined}
				onFilter={(group) => setSearchParams({ group })}
				onClear={(group) => {
					clear.mutate(group);
				}}
			/>
		{/if}
	{/if}

	<CacheKeyTable
		keys={rows}
		empty={data.search || data.kind || data.group
			? 'No key matches these filters.'
			: 'Nothing is kept in this database yet.'}
		onOpen={open}
	/>

	{#if listing.hasNextPage}
		<ShowMore
			shown={rows.length}
			loading={listing.isFetchingNextPage}
			onclick={() => listing.fetchNextPage()}
		/>
	{/if}

	<CacheKeyDialog database={data.database} key={opened} bind:open={dialogOpen} />

	<ConfirmDialog
		bind:open={flushing}
		title={`Flush the ${label.toLowerCase()} database?`}
		description={data.database === 'sessions'
			? 'Every session, administrator and client kept here is removed and read from the database again on the next request — nobody is signed out. Every rate limit starts counting again from nothing.'
			: 'Everything cached is removed, and each page reads the database again the first time it is asked for. Nobody notices but the next few pages, which are slower.'}
		confirmLabel={`Flush ${label.toLowerCase()}`}
		busy={flush.isPending}
		onConfirm={() => {
			flush.mutate();
		}}
	/>
{/if}

<style>
	.about {
		margin: 0;
		max-width: 60rem;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
