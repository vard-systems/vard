<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { RiRefreshLine } from 'svelte-remixicon';
	import { BRAND } from '#lib/brand.js';
	import { IconButton, PageContainer, PageHeader } from '#lib/components/ui/index.js';
	import OtpSettings from '#lib/components/otp/OtpSettings.svelte';
	import { keys, otpOptions } from '#lib/query/index.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const queryClient = useQueryClient();

	const settings = createQuery(() => otpOptions(data.otp));
</script>

<svelte:head><title>One-time codes · {BRAND.name}</title></svelte:head>

<PageHeader
	crumbs={['Dashboard', 'One-time codes']}
	description="The codes this server emails people as they sign in: how long one is, how long it lasts, and how many guesses it takes. The codes an authenticator app shows are not these — those are fixed by the apps that read them."
>
	{#snippet secondary()}
		<IconButton
			icon={RiRefreshLine}
			label="Reload these settings"
			onclick={() => queryClient.invalidateQueries({ queryKey: keys.otp.settings })}
		/>
	{/snippet}
</PageHeader>

<PageContainer>
	<OtpSettings settings={settings.data} />
</PageContainer>
