<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { RiAddLine, RiDeleteBinLine, RiRefreshLine } from 'svelte-remixicon';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { apisApi, type API } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import { apisOptions, firstPage, keys, LIST_PAGE_SIZE } from '#lib/query/index.js';
	import { can } from '#lib/permissions.js';
	import {
		Button,
		ConfirmDialog,
		Icon,
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

	async function apply(changes: { search?: string }) {
		const params = new SvelteURLSearchParams(page.url.search);

		for (const [key, value] of Object.entries(changes)) {
			if (value) params.set(key, value);
			else params.delete(key);
		}

		const query = params.toString();
		const path = resolve('/admin/(panel)/dashboard/apis');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(query ? `${path}?${query}` : path, { reset: false });
	}

	let timer: ReturnType<typeof setTimeout>;

	function debounced() {
		clearTimeout(timer);
		timer = setTimeout(() => apply({ search }), 250);
	}

	function openApi(api: API) {
		goto(resolve('/admin/(panel)/dashboard/apis/[id]', { id: api.id }));
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

<svelte:head><title>APIs · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'APIs']}
	count={apis.data.length}
	description="APIs represent the protected resources that applications can access. Applications request access tokens for an API, and each token is issued for a specific audience."
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
			<Button onclick={() => (dialogOpen = true)}>
				<Icon icon={RiAddLine} />
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
		onsubmit={() => apply({ search })}
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
		<Icon icon={RiDeleteBinLine} />
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
