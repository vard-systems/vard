<script lang="ts">
	import type { SSOConnection } from '#lib/api/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import { ssoHref } from '#lib/utils/links.js';
	import ProviderLink from './ProviderLink.svelte';

	type Props = {
		connections: SSOConnection[];
		request?: string | null;
		next?: string | null;
		disabled?: boolean;
	};

	let { connections, request = null, next = null, disabled = false }: Props = $props();

	const t = useTranslator();
</script>

{#if connections.length > 0}
	<div class="sso">
		{#each connections as connection (connection.slug)}
			<ProviderLink
				href={ssoHref(connection.slug, { request, next })}
				label={t('login.continue_with', { name: connection.name })}
				icon="ri-building-line"
				{disabled}
			/>
		{/each}
	</div>
{/if}

<style>
	.sso {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}
</style>
