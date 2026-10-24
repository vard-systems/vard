<script module lang="ts">
	/** The sentence for each scope the server grants; any other is shown by name. */
	const scopeKeys: Record<string, string> = {
		openid: 'apps.scope_openid',
		profile: 'apps.scope_profile',
		email: 'apps.scope_email',
		offline_access: 'apps.scope_offline',
		roles: 'apps.scope_roles'
	};
</script>

<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { account, type ConnectedApplication } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { formatDate, timeAgo } from '#lib/utils/format.js';
	import { AppMark } from '../auth/index.js';
	import { Button, Panel } from '../ui/index.js';

	type Props = {
		application: ConnectedApplication;
		timezone: string;
	};

	let { application: app, timezone }: Props = $props();

	const t = useTranslator();

	let confirming = $state(false);
	let disconnecting = $state(false);

	async function disconnect() {
		disconnecting = true;

		try {
			await account.disconnect(app.client_id);
			notify.success(t('apps.disconnected', { app: app.name }), t('apps.disconnected_body'));
			confirming = false;
			await refreshAll();
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			disconnecting = false;
		}
	}
</script>

<Panel title={app.name} description={t('apps.last_used', { when: timeAgo(app.last_used_at, t) })}>
	{#snippet aside()}
		<AppMark name={app.name} logo={app.logo_url} size={44} />
	{/snippet}

	<div>
		<h3>{t('apps.it_can')}</h3>
		<ul>
			{#each app.scopes as scope (scope)}
				<li>
					<i class="ri-check-line" style:font-size="16px" aria-hidden="true"></i>
					{scopeKeys[scope] ? t(scopeKeys[scope]) : scope}
				</li>
			{/each}
		</ul>
	</div>

	<p class="meta">
		{t('apps.connected_since', { when: formatDate(app.authorized_at, t, timezone) })}
		{#if app.website_url}
			·
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={app.website_url} target="_blank" rel="noopener noreferrer">
				{t('apps.visit')}
				<i class="ri-external-link-line" style:font-size="13px" aria-hidden="true"></i>
			</a>
		{/if}
	</p>

	{#snippet footer()}
		{#if confirming}
			<span class="confirm">{t('apps.disconnect_confirm', { app: app.name })}</span>
			<Button variant="secondary" size="sm" onclick={() => (confirming = false)}>
				{t('apps.keep')}
			</Button>
			<Button variant="danger" size="sm" loading={disconnecting} onclick={disconnect}>
				{t('apps.disconnect')}
			</Button>
		{:else}
			<Button variant="danger" size="sm" onclick={() => (confirming = true)}>
				{t('apps.disconnect')}
			</Button>
		{/if}
	{/snippet}
</Panel>

<style>
	h3 {
		margin: 0 0 var(--space-2);
		font-size: var(--text-sm);
		color: var(--color-text-hint);
		font-weight: 600;
	}

	ul {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: var(--space-2);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	li i {
		color: var(--color-success);
	}

	.meta {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.meta a {
		display: inline-flex;
		align-items: center;
		gap: 2px;
	}

	.confirm {
		margin-right: auto;
		align-self: center;
		font-weight: 600;
	}
</style>
