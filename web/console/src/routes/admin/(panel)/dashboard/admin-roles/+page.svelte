<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { RiAddLine, RiDeleteBinLine, RiRefreshLine } from 'svelte-remixicon';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { adminsApi, type AdminRole } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		adminPermissionsOptions,
		adminRolesOptions,
		firstPage,
		keys,
		LIST_PAGE_SIZE
	} from '#lib/query/index.js';
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
	import AdminRoleDialog from '#lib/components/admins/AdminRoleDialog.svelte';
	import AdminRoleTable from '#lib/components/admins/AdminRoleTable.svelte';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const roles = createQuery(() => adminRolesOptions(data.search, data.roles));
	const catalog = createQuery(() => adminPermissionsOptions(data.catalog));

	let shown = $derived(firstPage(data.search));
	const rows = $derived(roles.data.slice(0, shown));

	let search = $derived(data.search);

	let editing = $state<AdminRole | null>(null);
	let roleOpen = $state(false);

	let selected = $state<string[]>([]);

	const deletable = $derived(
		new Set(rows.filter((role) => !role.is_builtin).map((role) => role.id))
	);
	const chosen = $derived(selected.filter((id) => deletable.has(id)));

	const holders = $derived(
		roles.data
			.filter((role) => chosen.includes(role.id))
			.reduce((sum, role) => sum + role.admin_count, 0)
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
		const path = resolve('/admin/(panel)/dashboard/admin-roles');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(query ? `${path}?${query}` : path, { reset: false });
	}

	let timer: ReturnType<typeof setTimeout>;

	function debounced() {
		clearTimeout(timer);
		timer = setTimeout(() => apply({ search }), 250);
	}

	function openRole(role: AdminRole | null) {
		editing = role;
		roleOpen = true;
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
				await adminsApi.removeRole(id);
			}
		},
		onSuccess: (_result: void, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'admin role')} deleted`);
			reset();
			return queryClient.invalidateQueries({ queryKey: keys.admins.all });
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not delete these roles');
		},
		onSettled: () => {
			busy = false;
		}
	}));
</script>

<svelte:head><title>Admin roles · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Admin roles']}
	count={roles.data.length}
	description="What administrators may do in this panel. Each role grants permissions from a fixed catalog; managing administrators and these roles stays with super_admin."
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
		<Button onclick={() => openRole(null)}>
			<Icon icon={RiAddLine} />
			New admin role
		</Button>
	{/snippet}
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search admin roles"
		placeholder="Search name or description…"
		bind:value={search}
		onsubmit={() => apply({ search })}
		oninput={debounced}
	/>
</Toolbar>

<AdminRoleTable
	roles={rows}
	onOpen={openRole}
	selected={chosen}
	onSelect={(ids) => (selected = ids)}
/>

{#if roles.data.length > shown}
	<ShowMore {shown} total={roles.data.length} onclick={() => (shown += LIST_PAGE_SIZE)} />
{/if}

<SelectionBar count={chosen.length} onReset={reset}>
	<Button colorPalette="danger" size="sm" onclick={() => (confirmingDelete = true)} disabled={busy}>
		<Icon icon={RiDeleteBinLine} />
		Delete
	</Button>
</SelectionBar>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Delete ${chosen.length} ${chosen.length === 1 ? 'admin role' : 'admin roles'}?`}
	description={holders > 0
		? `${holders} ${holders === 1 ? 'administrator' : 'administrators'} will lose a role. This cannot be undone.`
		: 'Nobody holds them. This cannot be undone.'}
	confirmLabel={`Delete ${chosen.length}`}
	{busy}
	onConfirm={() => {
		if (busy) return;
		busy = true;
		removeSelected.mutate(chosen);
	}}
/>

<AdminRoleDialog role={editing} catalog={catalog.data} bind:open={roleOpen} />
