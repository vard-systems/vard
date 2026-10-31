<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';
	import { pageTitle } from '#lib/brand.js';
	import { Badge, CopyButton, Note, PageHeader, Tabs } from '#lib/components/ui/index.js';
	import { can } from '#lib/permissions.js';
	import { apiOptions } from '#lib/query/index.js';
	import ApiApplications from '#lib/components/apis/ApiApplications.svelte';
	import ApiIntrospectors from '#lib/components/apis/ApiIntrospectors.svelte';
	import ApiLogs from '#lib/components/apis/ApiLogs.svelte';
	import ApiOverview from '#lib/components/apis/ApiOverview.svelte';
	import ApiScopes from '#lib/components/apis/ApiScopes.svelte';
	import ApiSettings from '#lib/components/apis/ApiSettings.svelte';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const query = createQuery(() => apiOptions(data.api.id, data.api));
	const api = $derived(query.data);

	const editable = $derived(can(data.admin, 'apis.write'));

	let tab = $derived<string>(data.tab);

	function show(value: string) {
		tab = value;

		const url = new URL(page.url.href);
		if (value === 'overview') url.searchParams.delete('tab');
		else url.searchParams.set('tab', value);

		// The same page, with only its query changed.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto(url, { shallow: true, replace: true, state: page.state });
	}

	const tabs = $derived([
		{ value: 'overview', label: 'Overview', icon: 'ri-dashboard-line' },
		{ value: 'settings', label: 'Settings', icon: 'ri-settings-3-line' },
		{
			value: 'scopes',
			label: 'Scopes',
			icon: 'ri-lock-line',
			count: api.scopes.length
		},

		{
			value: 'applications',
			label: 'Applications',
			icon: 'ri-apps-line',
			count: api.application_count
		},
		{ value: 'logs', label: 'Logs', icon: 'ri-file-list-line' }
	]);
</script>

<svelte:head><title>{pageTitle(api.name, 'APIs')}</title></svelte:head>

<div class="page">
	<div class="heading">
		<PageHeader crumbs={['Dashboard', { label: 'APIs', href: resolve('/(panel)/apis') }, api.name]}>
			{#snippet meta()}
				{#if api.enforce_roles}
					<Badge tone="success">
						<i class="ri-shield-check-line" style:font-size="12px" aria-hidden="true"></i>
						Role-based access
					</Badge>
				{/if}
				{#if api.system}
					<Badge>Built-in</Badge>
				{/if}
				<Badge>{api.signing_algorithm}</Badge>
			{/snippet}
		</PageHeader>

		<div class="identifier">
			<code title="Identifier (audience)">{api.identifier}</code>
			<CopyButton value={api.identifier} label="identifier" />
		</div>

		{#if api.description}
			<p class="description">{api.description}</p>
		{/if}

		{#if api.system}
			<Note>
				{api.system === 'admin'
					? 'This is the server’s own admin API. Its scopes are the admin permissions: allow an application some of them — such as admin-cli — and its client credentials token can call the matching admin routes.'
					: 'This is the server’s own account API. An application allowed its scopes can call a signed-in user’s account routes with that user’s access token.'}
				Its identifier and scopes are the server’s, and it cannot be deleted.
			</Note>
		{/if}
	</div>

	<Tabs {tabs} bind:value={() => tab, show} label="API sections">
		{#snippet panel(value)}
			<div class="panel">
				{#if value === 'overview'}
					<ApiOverview {api} onTab={show} />
				{:else if value === 'settings'}
					<ApiSettings {api} {editable} />
				{:else if value === 'scopes'}
					<ApiScopes {api} editable={editable && !api.system} />
				{:else if value === 'applications'}
					<ApiApplications {api} admin={data.admin} />
					<ApiIntrospectors {api} {editable} />
				{:else if value === 'logs'}
					<ApiLogs {api} />
				{/if}
			</div>
		{/snippet}
	</Tabs>
</div>

<style>
	.page {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.heading {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.heading :global(.meta i) {
		margin-right: 2px;
		vertical-align: -1px;
	}

	.identifier {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		min-width: 0;
	}

	.identifier code {
		overflow-wrap: anywhere;
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.description {
		margin: var(--space-1) 0 0;
		max-width: 75ch;
		color: var(--color-text-hint);
		font-size: var(--text-base);
	}

	.panel {
		padding-bottom: var(--space-6);
	}
</style>
