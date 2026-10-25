<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { apisApi, type API } from '#lib/api/index.js';
	import { pageTitle } from '#lib/brand.js';
	import { apisOptions, firstPage, keys, LIST_PAGE_SIZE } from '#lib/query/index.js';
	import { can } from '#lib/permissions.js';
	import {
		Button,
		ConfirmDialog,
		IconButton,
		PageHeader,
		SearchInput,
		SelectionBar,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import ApiDialog from '#lib/components/apis/ApiDialog.svelte';
	import ApiTable from '#lib/components/apis/ApiTable.svelte';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const apis = createQuery(() => apisOptions(data.search, data.apis));

	let shown = $derived(firstPage(data.search));
	const rows = $derived(apis.data.slice(0, shown));

	const canWrite = $derived(can(data.admin, 'apis.write'));

	let search = $derived(data.search);

	let dialogOpen = $state(false);

	let selected = $state<string[]>([]);
	const visible = $derived(new Set(rows.map((api) => api.id)));
	const chosen = $derived(selected.filter((id) => visible.has(id)));

	const affected = $derived(
		apis.data
			.filter((api) => chosen.includes(api.id))
			.reduce((sum, api) => sum + api.application_count, 0)
	);

	let confirmingDelete = $state(false);
	let busy = $state(false);

	const debounced = debounce(() => setSearchParams({ search }));

	function openApi(api: API) {
		void goto(resolve('/(panel)/apis/[id]', { id: api.id }));
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.apis.all }),
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
				await apisApi.remove(id);
			}
		},
		onSuccess: (_result: void, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'API')} deleted`);
			reset();
			return Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.apis.all }),
				queryClient.invalidateQueries({ queryKey: keys.roles.all }),
				queryClient.invalidateQueries({ queryKey: keys.applications.all })
			]);
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not delete these APIs');
		},
		onSettled: () => {
			busy = false;
		}
	}));
</script>

<svelte:head><title>{pageTitle('APIs')}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'APIs']}
	count={apis.data.length}
	description="APIs represent the protected resources that applications can access. Applications request access tokens for an API, and each token is issued for a specific audience."
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
			<Button onclick={() => (dialogOpen = true)}>
				<i class="ri-add-line" aria-hidden="true"></i>
				New API
			</Button>
		{/if}
	{/snippet}
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search APIs"
		placeholder="Search name, identifier or description…"
		bind:value={search}
		onsubmit={() => setSearchParams({ search })}
		oninput={debounced}
	/>
</Toolbar>

<ApiTable
	apis={rows}
	onOpen={openApi}
	selected={canWrite ? chosen : undefined}
	onSelect={canWrite ? (ids) => (selected = ids) : undefined}
/>

{#if apis.data.length > shown}
	<ShowMore {shown} total={apis.data.length} onclick={() => (shown += LIST_PAGE_SIZE)} />
{/if}

<SelectionBar count={chosen.length} onReset={reset}>
	<Button colorPalette="danger" size="sm" onclick={() => (confirmingDelete = true)} disabled={busy}>
		<i class="ri-delete-bin-line" aria-hidden="true"></i>
		Delete
	</Button>
</SelectionBar>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Delete ${chosen.length} ${chosen.length === 1 ? 'API' : 'APIs'}?`}
	description={affected > 0
		? `${affected} ${affected === 1 ? 'application loses' : 'applications lose'} access, and roles lose these scopes. This cannot be undone.`
		: 'Roles lose these scopes. This cannot be undone.'}
	confirmLabel={`Delete ${chosen.length}`}
	{busy}
	onConfirm={() => {
		if (busy) return;
		busy = true;
		removeSelected.mutate(chosen);
	}}
/>

{#if canWrite}
	<ApiDialog bind:open={dialogOpen} />
{/if}
