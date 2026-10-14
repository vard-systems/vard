<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import { RiRefreshLine } from 'svelte-remixicon';
	import type { API, APILogEntry } from '#lib/api/index.js';
	import { Alert, Badge, IconButton, Note } from '#lib/components/ui/index.js';
	import { apiLogsOptions } from '#lib/query/index.js';
	import { formatDateTime } from '#lib/utils/format.js';

	type Props = {
		api: API;
	};

	let { api }: Props = $props();

	const logs = createQuery(() => apiLogsOptions(api.id));

	const actions: Record<string, { label: string; tone: 'neutral' | 'success' | 'danger' }> = {
		'api.created': { label: 'API created', tone: 'success' },
		'api.updated': { label: 'API updated', tone: 'neutral' },
		'api.deleted': { label: 'API deleted', tone: 'danger' },
		'application.api_authorized': { label: 'Access granted', tone: 'success' },
		'application.api_revoked': { label: 'Access revoked', tone: 'danger' }
	};

	function describe(entry: APILogEntry): string {
		const application = entry.metadata?.application;

		if (entry.action === 'application.api_authorized' && typeof application === 'string') {
			return `${application} may request tokens, or had its scopes changed`;
		}
		if (entry.action === 'application.api_revoked' && typeof application === 'string') {
			return `${application} may no longer request tokens`;
		}

		return '';
	}
</script>

<div class="head">
	<p class="lead">
		Changes to this API and to which applications may use it, newest first. Token requests appear
		here once the token endpoint issues tokens.
	</p>
	<IconButton
		icon={RiRefreshLine}
		label="Refresh the log"
		size="sm"
		loading={logs.isFetching}
		onclick={() => logs.refetch()}
	/>
</div>

{#if logs.isPending}
	<Note>Loading the log…</Note>
{:else if logs.isError}
	<Alert>Could not load the log.</Alert>
{:else if logs.data.length === 0}
	<p class="none">Nothing has happened to this API yet.</p>
{:else}
	<div class="scroll">
		<table>
			<thead>
				<tr>
					<th>Event</th>
					<th>Details</th>
					<th>By</th>
					<th>IP</th>
					<th class="end">When</th>
				</tr>
			</thead>
			<tbody>
				{#each logs.data as entry (entry.id)}
					{@const known = actions[entry.action]}
					<tr>
						<td>
							<Badge tone={known?.tone ?? 'neutral'}>{known?.label ?? entry.action}</Badge>
						</td>
						<td class="details">{describe(entry) || '—'}</td>
						<td>{entry.actor || 'System'}</td>
						<td><code>{entry.ip || '—'}</code></td>
						<td class="end nowrap">{formatDateTime(entry.created_at)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.head {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		margin-bottom: var(--space-3);
	}

	.lead {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.5;
	}

	.none {
		margin: 0;
		padding: var(--space-4);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-surface);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		text-align: center;
	}

	.scroll {
		overflow-x: auto;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	table {
		width: 100%;
		min-width: 40rem;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}

	th {
		padding: 8px var(--space-3);
		border-bottom: 1px solid var(--color-border);
		background: var(--color-secondary-alt);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
		font-weight: 700;
		text-align: left;
	}

	td {
		padding: 10px var(--space-3);
		vertical-align: middle;
	}

	tr + tr td {
		border-top: 1px solid var(--color-border);
	}

	.details {
		color: var(--color-text-hint);
	}

	.end {
		text-align: right;
	}

	.nowrap {
		white-space: nowrap;
	}

	code {
		font-family: var(--font-mono);
		font-size: var(--text-xs);
	}
</style>
