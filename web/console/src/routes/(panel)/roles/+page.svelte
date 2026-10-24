<script lang="ts">
	import {
		createInfiniteQuery,
		createMutation,
		createQuery,
		useQueryClient
	} from '@tanstack/svelte-query';
	import { rolesApi, type Role } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		applicationChoicesOptions,
		keys,
		roleChoicesOptions,
		rolesOptions,
		uniqueById
	} from '#lib/query/index.js';
	import { canEditRoles } from '#lib/components/roles/roles.js';
	import {
		Button,
		ConfirmDialog,
		IconButton,
		PageHeader,
		SearchInput,
		SegmentedControl,
		SelectionBar,
		Note,
		ShowMore,
		TabList,
		Toolbar,
		notify,
		type Tab
	} from '#lib/components/ui/index.js';
	import RoleDialog from '#lib/components/roles/RoleDialog.svelte';
	import RoleTable from '#lib/components/roles/RoleTable.svelte';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';
	import { datedName, saveJson } from '#lib/utils/download.js';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const roles = createInfiniteQuery(() =>
		rolesOptions(
			{
				scope: data.tab,
				application: data.application,
				search: data.search,
				isDefault: data.isDefault
			},
			data.page
		)
	);
	const rows = $derived(uniqueById(roles.data?.pages.flatMap((one) => one.roles) ?? []));
	const total = $derived(roles.data?.pages.at(-1)?.total ?? 0);
	const choices = createQuery(() => roleChoicesOptions(data.choices));
	const applications = createQuery(() => applicationChoicesOptions(data.applications));

	const globalCount = $derived(choices.data.filter((role) => role.application_id === null).length);
	const appCount = $derived(choices.data.length - globalCount);

	const kinds: Tab[] = $derived([
		{ value: 'global', label: 'Global roles', icon: 'ri-global-line', count: globalCount },
		...(applications.data.length > 0
			? [
					{
						value: 'application',
						label: 'Application roles',
						icon: 'ri-apps-line',
						count: appCount
					}
				]
			: [])
	]);

	/** Each kind starts unfiltered: a search for one is rarely the search for the other. */
	function showKind(kind: string) {
		void setSearchParams({
			tab: kind === 'global' ? '' : kind,
			application: '',
			search: '',
			default: ''
		});
	}

	const canCreate = $derived(
		canEditRoles(data.admin, null) ||
			applications.data.some((app) => canEditRoles(data.admin, app.id))
	);

	const canWrite = $derived(canEditRoles(data.admin, null));

	let search = $derived(data.search);

	let editing = $state<Role | null>(null);
	let roleOpen = $state(false);

	let selected = $state<string[]>([]);

	/** The ticked rows that are actually on screen: a search can take a
	    ticked row out of view, and deleting what nobody can see is not
	    something a panel should offer. */
	const visible = $derived(new Set(rows.map((role) => role.id)));
	const chosen = $derived(selected.filter((id) => visible.has(id)));
	let confirmingDelete = $state(false);

	let busy = $state(false);

	const holders = $derived(
		rows.filter((role) => chosen.includes(role.id)).reduce((sum, role) => sum + role.user_count, 0)
	);

	const debounced = debounce(() => setSearchParams({ search }));

	function openRole(role: Role | null) {
		editing = role;
		roleOpen = true;
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.roles.all }),
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

	function chosenRoles(): Role[] {
		return rows.filter((role) => chosen.includes(role.id));
	}

	const removeSelected = createMutation(() => ({
		mutationFn: async (ids: string[]) => {
			for (const id of ids) {
				await rolesApi.remove(id);
			}
		},
		onSuccess: (_result: void, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'role')} deleted`);
			reset();
			return Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.roles.all }),
				queryClient.invalidateQueries({ queryKey: keys.users.all })
			]);
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not delete these roles');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	function download() {
		saveJson(datedName('roles'), chosenRoles());
	}

	const filters = [
		{ value: '', label: 'All', reset: true },
		{ value: 'true', label: 'Default' },
		{ value: 'false', label: 'Not default' }
	];
</script>

<svelte:head><title>Roles · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Roles']}
	description="Roles say what users may do in your applications. A global role is the same everywhere; an application role only means something in its own application."
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
		{#if canCreate}
			<Button onclick={() => openRole(null)}>
				<i class="ri-add-line" aria-hidden="true"></i>
				New role
			</Button>
		{/if}
	{/snippet}
</PageHeader>

<TabList tabs={kinds} value={data.tab} label="Kinds of role" onchange={showKind} />

{#if data.tab === 'application' && data.application}
	<Note>
		Showing the roles of
		<strong>{applications.data.find((app) => app.id === data.application)?.name}</strong>.
		<button type="button" onclick={() => setSearchParams({ application: '' })}>
			Show every application
		</button>
	</Note>
{/if}

<Toolbar>
	<SearchInput
		label="Search roles"
		placeholder="Search roles…"
		bind:value={search}
		onsubmit={() => setSearchParams({ search })}
		oninput={debounced}
	/>

	<SegmentedControl
		label="Filter by default"
		options={filters}
		value={data.isDefault}
		onChange={(value) => setSearchParams({ default: value })}
	/>
</Toolbar>

<RoleTable
	roles={rows}
	showApplication={data.tab === 'application'}
	applications={applications.data}
	onOpen={openRole}
	selected={canWrite ? chosen : undefined}
	onSelect={canWrite ? (ids) => (selected = ids) : undefined}
/>

{#if roles.hasNextPage}
	<ShowMore
		shown={rows.length}
		{total}
		loading={roles.isFetchingNextPage}
		onclick={() => roles.fetchNextPage()}
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
	title={`Delete ${chosen.length} ${chosen.length === 1 ? 'role' : 'roles'}?`}
	description={holders > 0
		? `${holders} ${holders === 1 ? 'user' : 'users'} will lose a role. This cannot be undone.`
		: 'Nobody holds them. This cannot be undone.'}
	confirmLabel={`Delete ${chosen.length}`}
	{busy}
	onConfirm={() => {
		if (busy) return;
		busy = true;
		removeSelected.mutate(chosen);
	}}
/>

<RoleDialog
	role={editing}
	initialScope={data.tab === 'application'
		? data.application || (applications.data[0]?.id ?? null)
		: null}
	roles={choices.data}
	applications={applications.data}
	apis={data.apis}
	admin={data.admin}
	bind:open={roleOpen}
/>
