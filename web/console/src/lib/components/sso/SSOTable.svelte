<script lang="ts">
	import {
		RiBuilding2Line,
		RiGlobalLine,
		RiGroupLine,
		RiShieldKeyholeLine,
		RiToggleLine
	} from '#lib/icons.js';
	import type { SSOConnection } from '#lib/api/index.js';
	import { Badge, Code, type Column, DataTable, Icon, Tag } from '#lib/components/ui/index.js';

	type Props = {
		connections: SSOConnection[];
		onOpen: (connection: SSOConnection) => void;
		empty: string;
	};

	let { connections, onOpen, empty }: Props = $props();

	const columns = $derived<Column[]>([
		{ key: 'connection', label: 'Connection', icon: RiBuilding2Line, min: '14rem' },
		{ key: 'domains', label: 'Domains', icon: RiGlobalLine, min: '14rem' },
		{ key: 'users', label: 'Users', icon: RiGroupLine, min: '7rem' },
		{ key: 'status', label: 'Status', icon: RiToggleLine, min: '14rem' }
	]);
</script>

<DataTable {columns} rows={connections} {empty} {onOpen} label={(connection) => connection.name}>
	{#snippet row(connection)}
		<td>
			<span class="connection">
				<span class="mark"><Icon icon={RiShieldKeyholeLine} size="1.05rem" /></span>
				<span class="names">
					<strong>{connection.name}</strong>
					<span class="protocol">
						{connection.protocol === 'saml' ? 'SAML 2.0' : 'OpenID Connect'}
					</span>
				</span>
			</span>
		</td>

		<td>
			<span class="domains">
				{#each connection.domains.slice(0, 3) as domain (domain)}
					<Code tone="quiet">{domain}</Code>
				{/each}
				{#if connection.domains.length > 3}
					<span class="more">+{connection.domains.length - 3}</span>
				{/if}
			</span>
		</td>

		<td>
			{#if connection.users > 0}
				<span>{connection.users.toLocaleString()}</span>
			{:else}
				<span class="none">—</span>
			{/if}
		</td>

		<td>
			<span class="status">
				{#if connection.is_enabled}
					<Badge tone="success">on</Badge>
				{:else}
					<Badge>off</Badge>
				{/if}
				{#if connection.enforce_domains}
					<Tag tone="info" small>enforced</Tag>
				{/if}
				{#if connection.show_on_login}
					<Tag small>on sign-in page</Tag>
				{/if}
			</span>
		</td>
	{/snippet}
</DataTable>

<style>
	.connection {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.mark {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		width: 30px;
		height: 30px;
		border: 1px solid var(--color-secondary-alt);
		border-radius: var(--radius-sm);
		background: var(--color-surface-alt);
		color: var(--color-text-hint);
	}

	.names {
		display: flex;
		flex-direction: column;
		min-width: 0;
		line-height: 1.3;
	}

	.protocol {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.domains,
	.status {
		display: inline-flex;
		align-items: center;
		flex-wrap: wrap;
		gap: var(--space-1);
	}

	.more,
	.none {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
