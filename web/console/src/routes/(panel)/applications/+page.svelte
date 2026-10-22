<script lang="ts">
	import { createInfiniteQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { applicationsApi, type Application } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import { applicationsOptions, keys, uniqueById } from '#lib/query/index.js';
	import { can } from '#lib/permissions.js';
	import {
		Button,
		ConfirmDialog,
		IconButton,
		PageHeader,
		SearchInput,
		SegmentedControl,
		SelectionBar,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import ApplicationDialog from '#lib/components/applications/ApplicationDialog.svelte';
	import ApplicationTable from '#lib/components/applications/ApplicationTable.svelte';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const applications = createInfiniteQuery(() =>
		applicationsOptions({ search: data.search, type: data.type }, data.page)
	);
	const rows = $derived(
		uniqueById(applications.data?.pages.flatMap((one) => one.applications) ?? [])
	);
	const total = $derived(applications.data?.pages.at(-1)?.total ?? 0);

	const canRegister = $derived(can(data.admin, 'applications.write'));

	let search = $derived(data.search);

	let editing = $state<Application | null>(null);
	let dialogOpen = $state(false);

	let selected = $state<string[]>([]);
	const visible = $derived(new Set(rows.map((app) => app.id)));
	const chosen = $derived(selected.filter((id) => visible.has(id)));

	let confirmingDelete = $state(false);
	let busy = $state(false);

	const debounced = debounce(() => setSearchParams({ search }));

	function openApplication(application: Application | null) {
		editing = application;
		dialogOpen = true;
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.applications.all }),
				new Promise((done) => setTimeout(done, 400))
			]);
		} finally {
			refreshing = false;
		}
	}

	function reset() {
		selected = [];
		confirmingDelete = false;
	}

	const removeSelected = createMutation(() => ({
		mutationFn: async (ids: string[]) => {
			for (const id of ids) {
				await applicationsApi.remove(id);
			}
		},
		onSuccess: (_result: void, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'application')} deleted`);
			reset();
			return Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.applications.all }),
				queryClient.invalidateQueries({ queryKey: keys.roles.all }),
				queryClient.invalidateQueries({ queryKey: keys.users.all })
			]);
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not delete these applications');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	const filters = [
		{ value: '', label: 'All', reset: true },
		{ value: 'web', label: 'Web' },
		{ value: 'spa', label: 'SPA' },
		{ value: 'native', label: 'Native' },
		{ value: 'm2m', label: 'M2M' }
	];
</script>

<svelte:head><title>Applications · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Applications']}
	count={total}
	description="The apps and services that sign their users in here with OAuth 2.0 and OpenID Connect. Each defines its own roles, and a token for it carries its roles alone."
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
		{#if canRegister}
			<Button onclick={() => openApplication(null)}>
				<i class="ri-add-line" aria-hidden="true"></i>
				New application
			</Button>
		{/if}
	{/snippet}
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search applications"
		placeholder="Search name, description or client id…"
		bind:value={search}
		onsubmit={() => setSearchParams({ search })}
		oninput={debounced}
	/>

	<SegmentedControl
		label="Filter by type"
		options={filters}
		value={data.type}
		onChange={(type) => setSearchParams({ type })}
	/>
</Toolbar>

<ApplicationTable
	applications={rows}
	onOpen={openApplication}
	selected={canRegister ? chosen : undefined}
	onSelect={canRegister ? (ids) => (selected = ids) : undefined}
/>

{#if applications.hasNextPage}
	<ShowMore
		shown={rows.length}
		{total}
		loading={applications.isFetchingNextPage}
		onclick={() => applications.fetchNextPage()}
	/>
{/if}

<SelectionBar count={chosen.length} onReset={reset}>
	<Button colorPalette="danger" size="sm" onclick={() => (confirmingDelete = true)} disabled={busy}>
		<i class="ri-delete-bin-line" aria-hidden="true"></i>
		Delete
	</Button>
</SelectionBar>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Delete ${chosen.length} ${chosen.length === 1 ? 'application' : 'applications'}?`}
	description="Their roles, and everyone's hold on them, go too. This cannot be undone."
	confirmLabel={`Delete ${chosen.length}`}
	{busy}
	onConfirm={() => {
		if (busy) return;
		busy = true;
		removeSelected.mutate(chosen);
	}}
/>

<ApplicationDialog
	application={editing}
	editable={editing ? can(data.admin, 'applications.write', editing.id) : canRegister}
	admin={data.admin}
	bind:open={dialogOpen}
/>
