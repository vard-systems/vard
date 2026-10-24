<script lang="ts">
	import type { SSOConnection } from '#lib/api/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import { ssoHref } from '#lib/utils/links.js';

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
			<!-- The server's own path, which redirects to the provider: a full
			     navigation off this app, not a route of it. -->
			<!-- eslint-disable svelte/no-navigation-without-resolve -->
			<a
				class="provider"
				href={disabled ? undefined : ssoHref(connection.slug, { request, next })}
				aria-disabled={disabled}
				data-sveltekit-reload
			>
				<i class="ri-building-line" aria-hidden="true"></i>
				<span>{t('login.continue_with', { name: connection.name })}</span>
			</a>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		{/each}
	</div>
{/if}

<style>
	.sso {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin-top: var(--space-4);
	}

	.provider {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		min-height: 44px;
		padding: 0 var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text);
		font-size: var(--text-base);
		font-weight: 600;
		text-decoration: none;
	}

	.provider:hover {
		background: var(--color-surface-alt);
	}

	.provider span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.provider[aria-disabled='true'] {
		opacity: 0.6;
		pointer-events: none;
	}
</style>
