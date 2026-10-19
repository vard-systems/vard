<script lang="ts">
	import { useQueryClient } from '@tanstack/svelte-query';
	import { RiBuildingLine, RiRefreshLine, RiTranslate2 } from '#lib/icons.js';
	import { BRAND } from '#lib/brand.js';
	import { IconButton, PageContainer, PageHeader, Tabs } from '#lib/components/ui/index.js';
	import LanguageSettings from '#lib/components/languages/LanguageSettings.svelte';
	import OrganizationSettings from '#lib/components/organization/OrganizationSettings.svelte';
	import { can } from '#lib/permissions.js';
	import { keys } from '#lib/query/index.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const tabs = $derived([
		...(data.organization
			? [{ value: 'organization', label: 'Organization', icon: RiBuildingLine }]
			: []),
		...(data.languages ? [{ value: 'languages', label: 'Languages', icon: RiTranslate2 }] : [])
	]);

	// svelte-ignore state_referenced_locally
	let tab = $state(tabs[0].value);

	function reload() {
		void queryClient.invalidateQueries({ queryKey: keys.organization.settings });
		void queryClient.invalidateQueries({ queryKey: keys.languages.all });
	}
</script>

<svelte:head><title>Settings · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Settings']}
	description="Who this installation belongs to, and the languages its sign-in pages speak."
>
	{#snippet secondary()}
		<IconButton icon={RiRefreshLine} label="Reload these settings" onclick={reload} />
	{/snippet}
</PageHeader>

<Tabs {tabs} bind:value={tab} label="Settings">
	{#snippet panel(value)}
		{#if value === 'organization' && data.organization}
			<PageContainer size="lg">
				<OrganizationSettings
					initial={data.organization}
					editable={can(data.admin, 'organization.write')}
				/>
			</PageContainer>
		{:else if value === 'languages' && data.languages}
			<LanguageSettings initial={data.languages} canWrite={can(data.admin, 'languages.write')} />
		{/if}
	{/snippet}
</Tabs>
