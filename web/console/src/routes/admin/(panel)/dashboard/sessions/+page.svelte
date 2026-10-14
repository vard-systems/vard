<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { SvelteURLSearchParams } from 'svelte/reactivity';
	import { RiRefreshLine } from 'svelte-remixicon';
	import { createInfiniteQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { sessionsApi, type UserSessionRecord } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import { keys, sessionsOptions } from '#lib/query/index.js';
	import {
		ConfirmDialog,
		FilterChip,
		IconButton,
		PageHeader,
		SearchInput,
		ShowMore,
		Toolbar,
		notify
	} from '#lib/components/ui/index.js';
	import SessionTable from '#lib/components/sessions/SessionTable.svelte';
	import { can } from '#lib/permissions.js';
	import { countOf } from '#lib/utils/format.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const sessions = createInfiniteQuery(() =>
		sessionsOptions({ search: data.search, user: data.user }, data.page)
	);
	const rows = $derived(sessions.data?.pages.flatMap((one) => one.sessions) ?? []);

	const canWrite = $derived(can(data.admin, 'users.write'));

	let search = $derived(data.search);

	let confirming = $state<UserSessionRecord['user'] | null>(null);
	let asking = $state(false);

	async function apply(changes: { search?: string; user?: string; email?: string }) {
		const params = new SvelteURLSearchParams(page.url.search);

		for (const [key, value] of Object.entries(changes)) {
			if (value) params.set(key, value);
			else params.delete(key);
		}

		const query = params.toString();
		const path = resolve('/admin/(panel)/dashboard/sessions');

		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(query ? `${path}?${query}` : path, { reset: false });
	}

	let timer: ReturnType<typeof setTimeout>;

	function debounced() {
		clearTimeout(timer);
		timer = setTimeout(() => apply({ search }), 250);
	}

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;

		try {
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.sessions.all }),
				new Promise((done) => setTimeout(done, 400))
			]);
		} finally {
			refreshing = false;
		}
	}

	const end = createMutation(() => ({
		mutationFn: (session: UserSessionRecord) => sessionsApi.end(session.id),
		onSuccess: () => {
			notify.success('Session ended');
			return queryClient.invalidateQueries({ queryKey: keys.sessions.all });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not end this session');
		}
	}));

	const signOut = createMutation(() => ({
		mutationFn: (user: UserSessionRecord['user']) => sessionsApi.signOutUser(user.id),
		onSuccess: (result, user) => {
			asking = false;
			notify.success(
				`${user.email} is signed out everywhere`,
				`${countOf(result.sessions, 'session')} ended, ${countOf(result.tokens, 'application token')} revoked.`
			);
			return queryClient.invalidateQueries({ queryKey: keys.sessions.all });
		},
		onError: (err: unknown) => {
			asking = false;
			notify.error(err, 'Could not sign this user out');
		}
	}));
</script>

<svelte:head><title>Sessions · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Sessions']}
	count={rows.length}
	description="Everyone signed in right now, where from and since when. Signing a user out everywhere also revokes the tokens their applications hold — the step to take after a lost device or a compromised account."
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
</PageHeader>

<Toolbar>
	<SearchInput
		label="Search sessions"
		placeholder="Search by the start of an email…"
		bind:value={search}
		onsubmit={() => apply({ search })}
		oninput={debounced}
	/>

	{#if data.user}
		<FilterChip title="Show everyone's sessions" onclear={() => apply({ user: '', email: '' })}>
			{`Only ${data.email || data.user}`}
		</FilterChip>
	{/if}
</Toolbar>

<SessionTable
	sessions={rows}
	empty={data.search || data.user
		? 'No session belongs to an address starting like that.'
		: 'Nobody is signed in.'}
	onEnd={canWrite ? (session) => end.mutate(session) : undefined}
	onSignOutUser={canWrite
		? (session) => {
				confirming = session.user;
				asking = true;
			}
		: undefined}
	onUser={(session) => apply({ user: session.user.id, email: session.user.email, search: '' })}
	ending={end.isPending ? end.variables?.id : undefined}
/>

{#if sessions.hasNextPage}
	<ShowMore
		shown={rows.length}
		loading={sessions.isFetchingNextPage}
		onclick={() => sessions.fetchNextPage()}
	/>
{/if}

<ConfirmDialog
	bind:open={asking}
	title={`Sign ${confirming?.email ?? 'this user'} out everywhere?`}
	description="Every browser they are signed in on is signed out, and every token their applications hold is revoked. They can sign in again."
	tone="warning"
	confirmLabel="Sign out everywhere"
	busy={signOut.isPending}
	onConfirm={() => confirming && signOut.mutate(confirming)}
/>
