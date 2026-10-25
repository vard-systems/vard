<script lang="ts">
	import {
		createInfiniteQuery,
		createMutation,
		createQuery,
		useQueryClient
	} from '@tanstack/svelte-query';
	import { usersApi, type UserRecord } from '#lib/api/index.js';
	import { pageTitle } from '#lib/brand.js';
	import {
		applicationChoicesOptions,
		keys,
		roleChoicesOptions,
		uniqueById,
		userFieldsOptions,
		usersOptions
	} from '#lib/query/index.js';
	import { can } from '#lib/permissions.js';
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
	import FieldsDialog from '#lib/components/users/FieldsDialog.svelte';
	import UserDialog from '#lib/components/users/UserDialog.svelte';
	import UserTable from '#lib/components/users/UserTable.svelte';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';
	import { datedName, saveJson } from '#lib/utils/download.js';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const users = createInfiniteQuery(() =>
		usersOptions({ search: data.search, verified: data.verified, role: data.role }, data.page)
	);
	const rows = $derived(uniqueById(users.data?.pages.flatMap((one) => one.users) ?? []));
	const total = $derived(users.data?.pages.at(-1)?.total ?? 0);
	const fields = createQuery(() => userFieldsOptions(data.fields));
	const roles = createQuery(() => roleChoicesOptions(data.roles));
	const applications = createQuery(() => applicationChoicesOptions(data.applications));

	const canWrite = $derived(can(data.admin, 'users.write'));
	const canEditFields = $derived(can(data.admin, 'user_fields.write'));

	const filterRole = $derived(roles.data.find((role) => role.id === data.role));
	const filterApp = $derived(
		applications.data.find((app) => app.id === filterRole?.application_id)
	);

	let search = $derived(data.search);

	let editing = $state<UserRecord | null>(null);
	let userOpen = $state(false);
	let fieldsOpen = $state(false);

	let selected = $state<string[]>([]);

	/**
	 * Ticked rows that are actually on screen; the bulk bar acts only on these, never on rows a
	 * filter hid.
	 */
	const visible = $derived(new Set(rows.map((user) => user.id)));
	const chosen = $derived(selected.filter((id) => visible.has(id)));
	let confirmingDelete = $state(false);

	let busy = $state(false);

	const debounced = debounce(() => setSearchParams({ search }));

	function openUser(user: UserRecord | null) {
		editing = user;
		userOpen = true;
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.users.all }),
				queryClient.invalidateQueries({ queryKey: keys.roles.choices }),
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

	function chosenRecords(): UserRecord[] {
		return rows.filter((user) => chosen.includes(user.id));
	}

	const removeSelected = createMutation(() => ({
		mutationFn: async (ids: string[]) => {
			for (const id of ids) {
				await usersApi.remove(id);
			}
		},
		onSuccess: (_result: void, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'user')} deleted`);
			reset();
			return Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.users.all }),
				queryClient.invalidateQueries({ queryKey: keys.roles.all })
			]);
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not delete these users');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	function download() {
		saveJson(datedName('users'), chosenRecords());
	}

	const filters = [
		{ value: '', label: 'All' },
		{ value: 'true', label: 'Verified' },
		{ value: 'false', label: 'Unverified' }
	];
</script>

<svelte:head><title>{pageTitle('Users')}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Users']}
	count={total}
	description="Everyone with an account here, and the fields each of them carries."
>
	{#snippet secondary()}
		{#if canEditFields}
			<IconButton
				icon="ri-settings-3-line"
				label="Field settings"
				onclick={() => (fieldsOpen = true)}
			/>
		{/if}

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
			<Button onclick={() => openUser(null)}>
				<i class="ri-add-line" aria-hidden="true"></i>
				New user
			</Button>
		{/if}
	{/snippet}
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search users"
		placeholder="Search email or any field…"
		bind:value={search}
		onsubmit={() => setSearchParams({ search })}
		oninput={debounced}
	/>

	{#if data.role}
		<FilterChip title="Clear the role filter" onclear={() => setSearchParams({ role: '' })}>
			role: <strong>{filterRole?.name ?? 'unknown'}</strong>
			{#if filterApp}in {filterApp.name}{/if}
		</FilterChip>
	{/if}

	<SegmentedControl
		label="Filter by verified"
		options={filters}
		value={data.verified}
		onChange={(verified) => setSearchParams({ verified })}
	/>
</Toolbar>

<UserTable
	users={rows}
	fields={fields.data}
	applications={applications.data}
	onOpen={openUser}
	selected={canWrite ? chosen : undefined}
	onSelect={canWrite ? (ids) => (selected = ids) : undefined}
/>

{#if users.hasNextPage}
	<ShowMore
		shown={rows.length}
		{total}
		loading={users.isFetchingNextPage}
		onclick={() => users.fetchNextPage()}
	/>
{/if}

<SelectionBar count={chosen.length} onReset={reset}>
	<Button colorPalette="danger" size="sm" onclick={() => (confirmingDelete = true)} disabled={busy}>
		<i class="ri-delete-bin-line" aria-hidden="true"></i>
		Delete
	</Button>
	<Button size="sm" onclick={download} disabled={busy}>
		<i class="ri-download-line" aria-hidden="true"></i>
		JSON
	</Button>
</SelectionBar>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Delete ${chosen.length} ${chosen.length === 1 ? 'user' : 'users'}?`}
	description="They are deleted for good, with the sessions they have open and the roles they hold. This cannot be undone."
	confirmLabel={`Delete ${chosen.length}`}
	{busy}
	onConfirm={() => {
		if (busy) return;
		busy = true;
		removeSelected.mutate(chosen);
	}}
/>

<UserDialog
	user={editing}
	fields={fields.data}
	applications={applications.data}
	roles={roles.data}
	admin={data.admin}
	editable={canWrite}
	bind:open={userOpen}
/>
{#if canEditFields}
	<FieldsDialog fields={fields.data} bind:open={fieldsOpen} />
{/if}
