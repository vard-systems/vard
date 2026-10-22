<script lang="ts">
	import type { SocialProvider, SocialSpec } from '#lib/api/index.js';
	import { Badge, Code, type Column, DataTable, Tag } from '#lib/components/ui/index.js';
	import { hostOf, markFor, missingFrom } from './providers';

	type Props = {
		providers: SocialProvider[];
		kinds: SocialSpec[];
		onOpen: (provider: SocialProvider) => void;
		empty: string;
		selected?: string[];
		onSelect?: (ids: string[]) => void;
	};

	let { providers, kinds, onOpen, empty, selected, onSelect }: Props = $props();

	const columns: Column[] = [
		{ key: 'provider', label: 'Provider', min: '13rem', grow: true },
		{ key: 'client_id', label: 'Client ID', min: '16rem' },
		{ key: 'accounts', label: 'Accounts', min: '8rem' },
		{ key: 'status', label: 'Status', min: '11rem' }
	];

	function kindOf(provider: SocialProvider): SocialSpec | undefined {
		return kinds.find((kind) => kind.kind === provider.kind);
	}
</script>

<DataTable
	{columns}
	rows={providers}
	{empty}
	{onOpen}
	label={(provider) => `Edit ${provider.name}`}
	{selected}
	{onSelect}
>
	{#snippet row(provider)}
		{@const mark = markFor(provider.kind)}
		<td>
			<span class="provider">
				<span class="mark"><i class={mark} style:font-size="17px" aria-hidden="true"></i></span>
				<span class="names">
					<strong>{provider.name}</strong>
					<span class="slug">{provider.slug}</span>
				</span>
			</span>
		</td>

		<td><Code tone="quiet" truncate title={provider.client_id}>{provider.client_id}</Code></td>

		<td>
			{#if provider.identities > 0}
				<span class="text">{provider.identities.toLocaleString()}</span>
			{:else}
				<span class="empty">N/A</span>
			{/if}
		</td>

		<td>
			<span class="status">
				{#if missingFrom(provider, kindOf(provider))}
					<Tag tone="warning" strong>{missingFrom(provider, kindOf(provider))}</Tag>
				{:else if provider.is_enabled}
					<Badge tone="success">offered</Badge>
				{:else}
					<Badge>off</Badge>
				{/if}

				{#if !kindOf(provider)?.custom}
					<span class="host">{hostOf(kindOf(provider)?.authorize_url ?? '')}</span>
				{:else}
					<span class="host">{hostOf(provider.authorize_url)}</span>
				{/if}
			</span>
		</td>
	{/snippet}
</DataTable>

<style>
	.provider {
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

	.slug,
	.host {
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.status {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.host {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.empty {
		color: var(--color-text-disabled);
	}
</style>
