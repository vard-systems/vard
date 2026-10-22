<script lang="ts">
	import {
		createInfiniteQuery,
		createMutation,
		createQuery,
		useQueryClient
	} from '@tanstack/svelte-query';
	import { adminsApi, type AdminRecord } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		adminPermissionsOptions,
		adminRolesOptions,
		adminsOptions,
		applicationChoicesOptions,
		keys,
		uniqueById
	} from '#lib/query/index.js';
	import {
		Button,
		ConfirmDialog,
		FilterChip,
		IconButton,
		PageHeader,
		SearchInput,
		SegmentedControl,
		SelectionBar,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import AdminDialog from '#lib/components/admins/AdminDialog.svelte';
	import AdminTable from '#lib/components/admins/AdminTable.svelte';
	import SecurityPanel from '#lib/components/admins/SecurityPanel.svelte';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const admins = createInfiniteQuery(() =>
		adminsOptions({ search: data.search, status: data.status, role: data.role }, data.page)
	);
	const rows = $derived(uniqueById(admins.data?.pages.flatMap((one) => one.admins) ?? []));
	const total = $derived(admins.data?.pages.at(-1)?.total ?? 0);
	const roles = createQuery(() => adminRolesOptions('', data.roles));
	const catalog = createQuery(() => adminPermissionsOptions(data.catalog));
	const applications = createQuery(() => applicationChoicesOptions(data.applications));

	const filterRole = $derived(roles.data.find((role) => role.id === data.role));

	let search = $derived(data.search);

	let editing = $state<AdminRecord | null>(null);
	let adminOpen = $state(false);

	let selected = $state<string[]>([]);

	const deletable = $derived(
		new Set(rows.filter((admin) => admin.id !== data.admin.id).map((it) => it.id))
	);
	const chosen = $derived(selected.filter((id) => deletable.has(id)));

	let confirmingDelete = $state(false);
	let busy = $state(false);

	const debounced = debounce(() => setSearchParams({ search }));

	function openAdmin(admin: AdminRecord | null) {
		editing = admin;
		adminOpen = true;
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.admins.all }),
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
				await adminsApi.remove(id);
			}
		},
		onSuccess: (_result: void, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'administrator')} deleted`);
			reset();
			return queryClient.invalidateQueries({ queryKey: keys.admins.all });
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not delete these administrators');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	const filters = [
		{ value: '', label: 'All', reset: true },
		{ value: 'active', label: 'Active' },
		{ value: 'suspended', label: 'Suspended' },
		{ value: 'disabled', label: 'Disabled' }
	];
</script>

<svelte:head><title>Administrators · {BRAND.name}</title></svelte:head>

<PageHeader crumbs={['Dashboard', 'Administrators']} count={total}>
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
		<Button onclick={() => openAdmin(null)}>
			<i class="ri-add-line" aria-hidden="true"></i>
			New administrator
		</Button>
	{/snippet}
</PageHeader>

<SecurityPanel security={data.security} selfHasMFA={data.admin.mfa_enabled} />

<Toolbar>
	<SearchInput
		label="Search administrators"
		placeholder="Search email or name…"
		bind:value={search}
		onsubmit={() => setSearchParams({ search })}
		oninput={debounced}
	/>

	{#if data.role}
		<FilterChip title="Clear the role filter" onclear={() => setSearchParams({ role: '' })}>
			role: <strong>{filterRole?.name ?? 'unknown'}</strong>
		</FilterChip>
	{/if}

	<SegmentedControl
		label="Filter by status"
		options={filters}
		value={data.status}
		onChange={(status) => setSearchParams({ status })}
	/>
</Toolbar>

<AdminTable
	admins={rows}
	self={data.admin.id}
	onOpen={openAdmin}
	selected={chosen}
	onSelect={(ids) => (selected = ids)}
/>

{#if admins.hasNextPage}
	<ShowMore
		shown={rows.length}
		{total}
		loading={admins.isFetchingNextPage}
		onclick={() => admins.fetchNextPage()}
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
	title={`Delete ${chosen.length} ${chosen.length === 1 ? 'administrator' : 'administrators'}?`}
	description="Their sessions end and they can no longer sign in. This cannot be undone."
	confirmLabel={`Delete ${chosen.length}`}
	{busy}
	onConfirm={() => {
		if (busy) return;
		busy = true;
		removeSelected.mutate(chosen);
	}}
/>

<AdminDialog
	admin={editing}
	self={data.admin.id}
	roles={roles.data}
	applications={applications.data}
	catalog={catalog.data}
	bind:open={adminOpen}
/>
