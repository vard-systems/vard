<script lang="ts">
	import {
		RiAddLine,
		RiDeleteBinLine,
		RiEyeLine,
		RiEyeOffLine,
		RiRefreshLine
	} from '#lib/icons.js';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { socialApi, type SocialProvider } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import {
		Button,
		ConfirmDialog,
		Icon,
		IconButton,
		PageHeader,
		SearchInput,
		SegmentedControl,
		SelectionBar,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import SocialDialog from '#lib/components/social/SocialDialog.svelte';
	import SocialTable from '#lib/components/social/SocialTable.svelte';
	import { can } from '#lib/permissions.js';
	import { firstPage, keys, LIST_PAGE_SIZE, socialProvidersOptions } from '#lib/query/index.js';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';
	import { debounce, setSearchParams } from '#lib/utils/url.js';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const social = createQuery(() =>
		socialProvidersOptions({ providers: data.providers, kinds: data.kinds })
	);

	const canWrite = $derived(can(data.admin, 'social.write'));

	let search = $derived(data.search);

	const providers = $derived.by(() => {
		const term = data.search.trim().toLowerCase();

		return social.data.providers.filter((provider) => {
			if (data.status === 'enabled' && !provider.is_enabled) return false;
			if (data.status === 'off' && provider.is_enabled) return false;

			if (term === '') return true;

			return [provider.name, provider.slug, provider.kind, provider.client_id].some((value) =>
				value.toLowerCase().includes(term)
			);
		});
	});

	let shown = $derived(firstPage(data.search, data.status));
	const rows = $derived(providers.slice(0, shown));

	let editing = $state<SocialProvider | null>(null);
	let dialogOpen = $state(false);

	let selected = $state<string[]>([]);
	const visible = $derived(new Set(rows.map((provider) => provider.id)));
	const chosen = $derived(selected.filter((id) => visible.has(id)));

	let confirmingRemove = $state(false);
	let busy = $state(false);

	const chosenAccounts = $derived(
		social.data.providers
			.filter((provider) => chosen.includes(provider.id))
			.reduce((sum, provider) => sum + provider.identities, 0)
	);

	const debounced = debounce(() => setSearchParams({ search }));

	function open(provider: SocialProvider | null) {
		editing = provider;
		dialogOpen = true;
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.social.all }),
				new Promise((done) => setTimeout(done, 400))
			]);
		} finally {
			refreshing = false;
		}
	}

	function reset() {
		selected = [];
		confirmingRemove = false;
	}

	const setEnabled = createMutation(() => ({
		mutationFn: async (enabled: boolean) => {
			for (const id of chosen) {
				const provider = social.data.providers.find((one) => one.id === id);
				if (!provider || provider.is_enabled === enabled) continue;

				await socialApi.update(id, { is_enabled: enabled });
			}
		},
		onSuccess: (_result: void, enabled: boolean) => {
			notify.success(enabled ? 'Providers turned on' : 'Providers turned off');
			reset();
			return queryClient.invalidateQueries({ queryKey: keys.social.all });
		},
		onError: (err: unknown) => {
			confirmingRemove = false;
			notify.error(err, 'Could not change these providers');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	const removeSelected = createMutation(() => ({
		mutationFn: async (ids: string[]) => {
			for (const id of ids) {
				await socialApi.remove(id);
			}
		},
		onSuccess: (_result: void, ids: string[]) => {
			notify.success(`${countOf(ids.length, 'provider')} removed`);
			reset();
			return Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.social.all }),
				queryClient.invalidateQueries({ queryKey: keys.users.all })
			]);
		},
		onError: (err: unknown) => {
			confirmingRemove = false;
			notify.error(err, 'Could not remove these providers');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	function run(what: 'enable' | 'disable' | 'remove') {
		if (busy) return;

		busy = true;

		if (what === 'remove') {
			removeSelected.mutate(chosen);
			return;
		}

		setEnabled.mutate(what === 'enable');
	}

	const filters = [
		{ value: '', label: 'All', reset: true },
		{ value: 'enabled', label: 'Offered' },
		{ value: 'off', label: 'Off' }
	];
</script>

<svelte:head><title>Social · {BRAND.name}</title></svelte:head>

<PageHeader crumbs={['Dashboard', 'Social']} count={social.data.providers.length}>
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
			<Button onclick={() => open(null)}>
				<Icon icon={RiAddLine} />
				Add provider
			</Button>
		{/if}
	{/snippet}
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search providers"
		placeholder="Search name, identifier or client id…"
		bind:value={search}
		onsubmit={() => setSearchParams({ search })}
		oninput={debounced}
	/>

	<SegmentedControl
		label="Filter by whether it is offered"
		options={filters}
		value={data.status}
		onChange={(status) => setSearchParams({ status })}
	/>
</Toolbar>

<SocialTable
	providers={rows}
	kinds={social.data.kinds}
	onOpen={open}
	empty={social.data.providers.length === 0
		? 'No providers yet. Add one to offer a button beside the password form.'
		: 'No providers match this.'}
	selected={canWrite ? chosen : undefined}
	onSelect={canWrite ? (ids) => (selected = ids) : undefined}
/>

{#if providers.length > shown}
	<ShowMore {shown} total={providers.length} onclick={() => (shown += LIST_PAGE_SIZE)} />
{/if}

<SelectionBar count={chosen.length} onReset={reset}>
	<Button size="sm" variant="subtle" onclick={() => run('enable')} disabled={busy}>
		<Icon icon={RiEyeLine} />
		Offer
	</Button>
	<Button size="sm" variant="subtle" onclick={() => run('disable')} disabled={busy}>
		<Icon icon={RiEyeOffLine} />
		Turn off
	</Button>
	<Button colorPalette="danger" size="sm" onclick={() => (confirmingRemove = true)} disabled={busy}>
		<Icon icon={RiDeleteBinLine} />
		Remove
	</Button>
</SelectionBar>

<ConfirmDialog
	bind:open={confirmingRemove}
	title={`Remove ${chosen.length} ${chosen.length === 1 ? 'provider' : 'providers'}?`}
	description={chosenAccounts > 0
		? `${chosenAccounts} ${chosenAccounts === 1 ? 'person signs' : 'people sign'} in with these. Their accounts stay; this way in goes.`
		: 'Nobody signs in with these yet.'}
	confirmLabel={`Remove ${chosen.length}`}
	{busy}
	onConfirm={() => run('remove')}
/>

<SocialDialog bind:open={dialogOpen} provider={editing} kinds={social.data.kinds} />
