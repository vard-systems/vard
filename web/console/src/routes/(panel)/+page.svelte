<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import type { OverviewDays } from '#lib/api/index.js';
	import { countOf } from '#lib/utils/format.js';
	import { BRAND } from '#lib/brand.js';
	import { logsHref } from '#lib/components/activity/filters.js';
	import ActivityChart from '#lib/components/activity/ActivityChart.svelte';
	import ActivityFeed from '#lib/components/activity/ActivityFeed.svelte';
	import SignInHealth from '#lib/components/activity/SignInHealth.svelte';
	import TopActors from '#lib/components/activity/TopActors.svelte';
	import SessionList from '#lib/components/profile/SessionList.svelte';
	import {
		IconButton,
		LinkButton,
		Note,
		PageHeader,
		Panel,
		SegmentedControl,
		StatCard,
		Tag
	} from '#lib/components/ui/index.js';
	import { can, canAnywhere } from '#lib/permissions.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const admin = $derived(data.admin);
	const overview = $derived(data.overview);

	const ranges: { value: `${OverviewDays}`; label: string }[] = [
		{ value: '7', label: '7 days' },
		{ value: '14', label: '14 days' },
		{ value: '30', label: '30 days' },
		{ value: '90', label: '90 days' }
	];

	async function pick(days: string) {
		const path = resolve('/(panel)');
		// resolve() has already applied any base path; the query is only
		// appended to what it returned.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(days === '14' ? path : `${path}?days=${days}`, { reset: false });
	}

	const logsPage = resolve('/(panel)/logs');

	const rangeStart = $derived(overview?.daily[0]?.day ?? '');

	const links = $derived({
		day: (day: string) => logsHref(logsPage, { from: day, to: day }),
		actor: (actor: string) => logsHref(logsPage, { actor, from: rangeStart }),
		refused: logsHref(logsPage, { kind: 'refused', from: rangeStart }),
		access: logsHref(logsPage, { kind: 'access', from: rangeStart })
	});

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;
		try {
			await Promise.all([refreshAll(), new Promise((done) => setTimeout(done, 400))]);
		} finally {
			refreshing = false;
		}
	}

	const totals = $derived(
		overview
			? {
					events: overview.daily.reduce((sum, day) => sum + day.events, 0),
					refused: overview.daily.reduce((sum, day) => sum + day.failures, 0)
				}
			: { events: 0, refused: 0 }
	);

	const count = (n: number) => n.toLocaleString();

	function note(...parts: (string | false)[]) {
		return parts.filter(Boolean).join(' · ');
	}

	const notes = $derived.by(() => {
		if (!overview) return null;

		const counts = overview.counts;

		return {
			users: note(
				`${count(counts.active_users)} active`,
				counts.new_users > 0 && `+${count(counts.new_users)} in ${data.days} days`
			),
			applications: note(
				`${count(counts.enabled_applications)} enabled`,
				countOf(counts.user_roles, 'role')
			),
			apis: note(countOf(counts.api_scopes, 'scope')),
			sessions: note(
				countOf(counts.admins, 'admin'),
				counts.locked_admins > 0 && `${counts.locked_admins} locked`
			)
		};
	});

	const sessionsTone = $derived(
		overview && overview.counts.locked_admins > 0 ? 'warning' : 'neutral'
	);
</script>

<svelte:head>
	<title>Activity · {BRAND.name}</title>
</svelte:head>

<div class="page">
	<PageHeader
		crumbs={['Dashboard', 'Activity']}
		description={overview
			? `What has been happening over the last ${data.days} days. Every number opens the entries behind it.`
			: undefined}
	>
		{#snippet secondary()}
			<IconButton
				icon="ri-refresh-line"
				label="Refresh"
				onclick={refresh}
				loading={refreshing}
				disabled={refreshing}
			/>
		{/snippet}

		{#snippet actions()}
			{#if overview}
				<SegmentedControl label="Range" options={ranges} value={`${data.days}`} onChange={pick} />
				<LinkButton href={resolve('/(panel)/logs')} variant="subtle">
					<i class="ri-file-list-3-line" aria-hidden="true"></i>
					Logs
				</LinkButton>
			{/if}
		{/snippet}
	</PageHeader>

	{#if overview && notes}
		{@const counts = overview.counts}
		<div class="stats">
			<StatCard
				label="Users"
				value={counts.users}
				icon="ri-group-line"
				note={notes.users}
				href={can(admin, 'users.read') ? resolve('/(panel)/users') : undefined}
			/>

			<StatCard
				label="Applications"
				value={counts.applications}
				icon="ri-apps-line"
				note={notes.applications}
				href={canAnywhere(admin, 'applications.read')
					? resolve('/(panel)/applications')
					: undefined}
			/>

			<StatCard
				label="APIs"
				value={counts.apis}
				icon="ri-code-box-line"
				note={notes.apis}
				href={can(admin, 'apis.read') ? resolve('/(panel)/apis') : undefined}
			/>

			<StatCard
				label="Active sessions"
				value={counts.active_sessions}
				icon="ri-computer-line"
				note={notes.sessions}
				tone={sessionsTone}
				href={admin.is_super_admin ? resolve('/(panel)/admins') : undefined}
			/>
		</div>

		<div class="row">
			<Panel title="Activity · last {data.days} days" icon="ri-bar-chart-box-line">
				{#snippet meta()}
					<Tag>{countOf(totals.events, 'event')}</Tag>
					{#if totals.refused > 0}
						<Tag tone="danger" dot>{totals.refused.toLocaleString()} refused</Tag>
					{/if}
				{/snippet}
				<ActivityChart daily={overview.daily} href={links.day} />
			</Panel>

			<Panel title="Sign-ins · last {data.days} days" icon="ri-shield-check-line">
				<SignInHealth
					signIns={overview.sign_ins}
					locked={counts.locked_admins}
					hrefs={{ succeeded: links.access, failed: links.refused, blocked: links.refused }}
				/>
			</Panel>
		</div>

		<div class="row">
			<Panel title="Recent activity" icon="ri-history-line" flush>
				{#snippet meta()}
					<LinkButton href={resolve('/(panel)/logs')} variant="ghost" size="sm">
						View all
					</LinkButton>
				{/snippet}
				<ActivityFeed events={overview.activity} />
			</Panel>

			<div class="side">
				<Panel title="Most active · last {data.days} days" icon="ri-user-star-line" flush>
					<TopActors actors={overview.top_actors} href={links.actor} />
				</Panel>

				<Panel title="Your sessions" icon="ri-computer-line" flush>
					<SessionList sessions={data.sessions} limit={4} />
				</Panel>
			</div>
		</div>
	{:else}
		<div class="narrow">
			<Panel title="Your sessions" icon="ri-computer-line" flush>
				<SessionList sessions={data.sessions} limit={8} />
			</Panel>

			<Note>
				Your roles do not include reading activity, so the numbers and the log are not shown here.
			</Note>
		</div>
	{/if}
</div>

<style>
	.page {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.stats {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: var(--space-3);
	}

	.row {
		display: grid;
		grid-template-columns: minmax(0, 1.7fr) minmax(0, 1fr);
		align-items: stretch;
		gap: var(--space-4);
	}

	.side {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		min-width: 0;
	}

	.narrow {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		max-width: 32rem;
	}

	@media (max-width: 80rem) {
		.stats {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 34rem) {
		.stats {
			grid-template-columns: minmax(0, 1fr);
		}
	}

	@media (max-width: 64rem) {
		.row {
			grid-template-columns: minmax(0, 1fr);
		}
	}

	@media (max-width: 34rem) {
		.row,
		.side,
		.page {
			gap: var(--space-3);
		}
	}
</style>
