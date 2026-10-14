<script lang="ts">
	import { resolve } from '$app/paths';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { RiAppsLine } from 'svelte-remixicon';
	import { applicationsApi, type Admin, type API, type APIApplication } from '#lib/api/index.js';
	import {
		Alert,
		Badge,
		Checkbox,
		Icon,
		Note,
		SearchInput,
		Switch,
		notify
	} from '#lib/components/ui/index.js';
	import { can } from '#lib/permissions.js';
	import { keys, apiApplicationsOptions } from '#lib/query/index.js';
	import { types } from '#lib/components/applications/applications.js';

	type Props = {
		api: API;
		admin: Admin;
	};

	let { api, admin }: Props = $props();

	const queryClient = useQueryClient();

	const applications = createQuery(() => apiApplicationsOptions(api.id));

	let search = $state('');

	let saving = $state<string | null>(null);

	const shown = $derived.by(() => {
		const all = applications.data ?? [];
		const term = search.trim().toLowerCase();
		const matching = term
			? all.filter((app) => app.name.toLowerCase().includes(term) || app.client_id.includes(term))
			: all;

		return [...matching].sort((a, b) => Number(b.authorized) - Number(a.authorized));
	});

	const authorized = $derived((applications.data ?? []).filter((app) => app.authorized).length);

	/** Saved as they are made, through the application's own endpoints, which
	    check the administrator may change that application. */
	const change = createMutation(() => ({
		mutationFn: ({ app, scopes }: { app: APIApplication; scopes: string[] | null }) =>
			scopes === null
				? applicationsApi.revokeAPI(app.id, api.id)
				: applicationsApi.authorizeAPI(app.id, api.id, scopes),
		onSuccess: async (_result, { app, scopes }) => {
			notify.success(
				scopes === null
					? `${app.name} can no longer call this API`
					: app.authorized
						? `Access saved for ${app.name}`
						: `${app.name} can now call this API`
			);
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.apis.applications(api.id) }),
				queryClient.invalidateQueries({ queryKey: keys.apis.one(api.id) }),
				queryClient.invalidateQueries({ queryKey: keys.apis.logs(api.id) }),
				queryClient.invalidateQueries({ queryKey: keys.apis.list('') }),
				queryClient.invalidateQueries({ queryKey: keys.applications.access(app.id) })
			]);
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not change this access');
		},
		onSettled: () => {
			saving = null;
		}
	}));

	function save(app: APIApplication, scopes: string[] | null) {
		if (saving) return;

		saving = app.id;
		change.mutate({ app, scopes });
	}

	function toggleScope(app: APIApplication, id: string, on: boolean) {
		save(app, on ? [...app.allowed, id] : app.allowed.filter((it) => it !== id));
	}
</script>

<p class="lead">
	The applications that may request access tokens for this API, and the most each token may carry.
	Authorising here is the same as on the application's API access tab.
</p>

{#if applications.isPending}
	<Note>Loading applications…</Note>
{:else if applications.isError}
	<Alert>Could not load the applications.</Alert>
{:else if applications.data.length === 0}
	<div class="empty">
		<Icon icon={RiAppsLine} size="1.25rem" />
		<div>
			<strong>No applications to authorise</strong>
			<p>
				Register one on the
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={resolve('/admin/(panel)/dashboard/applications')}>Applications page</a>, then allow
				it to use this API here.
			</p>
		</div>
	</div>
{:else}
	<div class="toolbar">
		<p class="summary">
			<strong>{authorized}</strong> of {applications.data.length}
			{applications.data.length === 1 ? 'application' : 'applications'} authorised
		</p>

		{#if applications.data.length > 5}
			<div class="search-slot">
				<SearchInput
					label="Filter applications"
					placeholder="Filter applications…"
					bind:value={search}
					size="sm"
				/>
			</div>
		{/if}
	</div>

	<div class="list">
		{#each shown as app (app.id)}
			{@const editable = can(admin, 'applications.write', app.id)}
			{@const kind = types[app.type]}
			<section class:authorized={app.authorized} class:busy={saving === app.id}>
				<header>
					<span class="kind" title={kind.label}><Icon icon={kind.icon} size="1rem" /></span>

					<div class="identity">
						<span class="name">
							<strong>{app.name}</strong>
							{#if !app.is_enabled}<Badge tone="danger">Disabled</Badge>{/if}
						</span>
						<code>{app.client_id}</code>
					</div>

					<Switch
						label={app.authorized ? 'Authorised' : 'Not authorised'}
						checked={app.authorized}
						disabled={!editable || saving !== null}
						onChange={(on) => save(app, on ? [] : null)}
					/>
				</header>

				{#if app.authorized && api.scopes.length > 0}
					<div class="scopes">
						<span class="count">
							{app.allowed.length} of {api.scopes.length} allowed
						</span>
						<ul>
							{#each api.scopes as scope (scope.id)}
								<li>
									<Checkbox
										checked={app.allowed.includes(scope.id)}
										onChange={(on) => toggleScope(app, scope.id, on)}
										disabled={!editable || saving !== null}
										title={scope.name}
									/>
									<code>{scope.name}</code>
									{#if scope.is_default}<Badge>default</Badge>{/if}
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			</section>
		{:else}
			<Note>No application matches “{search}”.</Note>
		{/each}
	</div>
{/if}

<style>
	.search-slot {
		display: flex;
		width: min(18rem, 100%);
	}

	.lead {
		margin: 0 0 var(--space-4);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.5;
	}

	.toolbar {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-2);
		margin-bottom: var(--space-3);
	}

	.summary {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.summary strong {
		color: var(--color-text);
	}

	.list {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	section {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		transition: opacity var(--speed-fast);
	}

	section.authorized {
		border-color: color-mix(in srgb, var(--color-success), var(--color-border) 55%);
	}

	section.busy {
		opacity: 0.6;
	}

	header {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-3);
	}

	.kind {
		display: grid;
		flex-shrink: 0;
		place-items: center;
		width: 32px;
		height: 32px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
		color: var(--color-text-hint);
	}

	.identity {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
	}

	.name {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	code {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}

	.identity code {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.scopes {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3) var(--space-3);
		border-top: 1px solid var(--color-border);
	}

	.count {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	ul {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(13rem, 1fr));
		gap: var(--space-2) var(--space-4);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.empty {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-4);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-surface);
		color: var(--color-text-hint);
	}

	.empty strong {
		color: var(--color-text);
	}

	.empty p {
		margin: 2px 0 0;
		font-size: var(--text-sm);
	}

	.empty a {
		color: var(--color-text);
	}
</style>
