<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import type { SSOConnection } from '#lib/api/index.js';
	import { pageTitle } from '#lib/brand.js';
	import {
		Button,
		EmptyState,
		IconButton,
		PageHeader,
		SearchInput,
		ShowMore,
		Toolbar
	} from '#lib/components/ui/index.js';
	import SSODialog from '#lib/components/sso/SSODialog.svelte';
	import SSOTable from '#lib/components/sso/SSOTable.svelte';
	import { can } from '#lib/permissions.js';
	import { firstPage, keys, LIST_PAGE_SIZE, ssoOptions } from '#lib/query/index.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const list = createQuery(() => ssoOptions({ connections: data.connections }));

	const canWrite = $derived(can(data.admin, 'sso.write'));

	let search = $state('');

	const visible = $derived.by(() => {
		const term = search.trim().toLowerCase();
		if (term === '') return list.data.connections;

		return list.data.connections.filter((connection) =>
			[connection.name, connection.slug, ...connection.domains].some((value) =>
				value.toLowerCase().includes(term)
			)
		);
	});

	let shown = $derived(firstPage(search));
	const rows = $derived(visible.slice(0, shown));

	let editing = $state<SSOConnection | null>(null);
	let dialogOpen = $state(false);

	function open(connection: SSOConnection | null) {
		editing = connection;
		dialogOpen = true;
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.sso.all }),
				new Promise((done) => setTimeout(done, 400))
			]);
		} finally {
			refreshing = false;
		}
	}
</script>

<svelte:head><title>{pageTitle('SSO integrations')}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'SSO integrations']}
	count={list.data.connections.length}
	description="Let an organisation sign its people in through its own identity provider — Okta, Microsoft Entra ID, Google Workspace, ADFS — over OpenID Connect or SAML 2.0. A connection owns email domains: the people at them sign in through it, can be made to, and get accounts and roles from what the provider says."
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
			<Button onclick={() => open(null)}>
				<i class="ri-add-line" aria-hidden="true"></i>
				New connection
			</Button>
		{/if}
	{/snippet}
</PageHeader>

{#if list.data.connections.length === 0}
	<EmptyState icon="ri-building-2-line" title="No identity providers connected yet" size="lg">
		Connect a company's identity provider and its people sign in with their work account: routed
		there by their email domain, given an account the first time, and roles from the groups the
		provider puts them in.

		{#snippet actions()}
			{#if canWrite}
				<Button onclick={() => open(null)}>
					<i class="ri-add-line" aria-hidden="true"></i>
					New connection
				</Button>
			{/if}
		{/snippet}
	</EmptyState>
{:else}
	<Toolbar>
		<SearchInput
			label="Search"
			placeholder="Search name, identifier or domain…"
			bind:value={search}
		/>
	</Toolbar>

	<SSOTable connections={rows} onOpen={open} empty="No connections match this." />

	{#if visible.length > shown}
		<ShowMore {shown} total={visible.length} onclick={() => (shown += LIST_PAGE_SIZE)} />
	{/if}
{/if}

<SSODialog
	bind:open={dialogOpen}
	connection={editing}
	roles={data.roles}
	canReadRoles={data.mayReadRoles}
	{canWrite}
/>
