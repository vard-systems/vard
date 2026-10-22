<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { BRAND } from '#lib/brand.js';
	import { IconButton, PageContainer, PageHeader, Tabs } from '#lib/components/ui/index.js';
	import MailContent from '#lib/components/mail/MailContent.svelte';
	import MailServer from '#lib/components/mail/MailServer.svelte';
	import { keys, mailContentOptions, mailOptions } from '#lib/query/index.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const settings = createQuery(() => mailOptions(data.mail));
	const content = createQuery(() => mailContentOptions(data.content));

	let tab = $state('server');

	const tabs = [
		{ value: 'server', label: 'Mail server', icon: 'ri-server-line' },
		{ value: 'content', label: 'Email content', icon: 'ri-file-text-line' }
	];

	function reload() {
		void queryClient.invalidateQueries({ queryKey: keys.mail.all });
	}
</script>

<svelte:head><title>Mail · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'Mail']}
	description="How this installation sends email — the server it hands a message to, and what each message says. Password resets, address confirmations and one-time codes all go this way."
>
	{#snippet secondary()}
		<IconButton icon="ri-refresh-line" label="Reload these settings" onclick={reload} />
	{/snippet}
</PageHeader>

<Tabs {tabs} bind:value={tab} label="Mail settings">
	{#snippet panel(value)}
		<PageContainer size="lg">
			{#if value === 'server'}
				<MailServer settings={settings.data} />
			{:else}
				<MailContent content={content.data} />
			{/if}
		</PageContainer>
	{/snippet}
</Tabs>
