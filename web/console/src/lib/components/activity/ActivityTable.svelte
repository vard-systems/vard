<script lang="ts">
	import { RiFileList3Line, RiMapPinLine, RiPulseLine, RiTimeLine } from 'svelte-remixicon';
	import type { LogEntry } from '#lib/api/index.js';
	import { DataTable, Tag, type Column } from '#lib/components/ui/index.js';
	import { describeUserAgent, formatDateTime, formatRelative } from '#lib/utils/format.js';
	import { describe } from './actions';

	type Props = {
		events: LogEntry[];
		empty?: string;
	};

	let { events, empty = 'Nothing has happened yet.' }: Props = $props();

	const columns: Column[] = [
		{ key: 'event', label: 'Event', icon: RiPulseLine, min: '12rem' },
		{ key: 'what', label: 'What happened', icon: RiFileList3Line, min: '22rem' },
		{ key: 'from', label: 'From', icon: RiMapPinLine, min: '11rem' },
		{ key: 'when', label: 'When', icon: RiTimeLine, min: '8rem', align: 'end' }
	];
</script>

<DataTable {columns} rows={events} {empty}>
	{#snippet row(event)}
		{@const said = describe(event)}
		<td>
			<Tag tone={said.tone} dot strong>{said.label}</Tag>
		</td>
		<td>
			<span class="sentence">
				<b>{said.actor}</b>
				{said.verb}
				{#if said.subject}<span class:name={said.named}>{said.subject}</span>{/if}
				{said.after}
			</span>
			{#if said.detail}<span class="detail {said.tone}">{said.detail}</span>{/if}
		</td>
		<td>
			<span class="from">
				<code>{event.ip || '—'}</code>
				{#if event.user_agent}
					<small title={event.user_agent}>{describeUserAgent(event.user_agent)}</small>
				{/if}
			</span>
		</td>
		<td class="end">
			<time datetime={event.created_at} title={formatDateTime(event.created_at)}>
				{formatRelative(event.created_at)}
			</time>
		</td>
	{/snippet}
</DataTable>

<style>
	.sentence {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.sentence b,
	.name {
		color: var(--color-text);
		font-weight: 600;
	}

	.detail {
		display: block;
		margin-top: 2px;
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.detail.danger {
		color: var(--color-danger);
	}

	.from {
		display: flex;
		flex-direction: column;
	}

	code {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	small {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	time {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		white-space: nowrap;
	}
</style>
