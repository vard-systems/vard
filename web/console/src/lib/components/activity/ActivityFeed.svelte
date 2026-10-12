<script lang="ts">
	import type { ActivityEvent } from '$lib/api';
	import { formatDateTime, formatDayHeading, formatTime } from '$lib/utils/format';
	import { Tag } from '$lib/components/ui';
	import { describe } from './actions';

	type Props = {
		events: ActivityEvent[];
		empty?: string;
	};

	let { events, empty = 'Nothing has happened yet.' }: Props = $props();

	const days = $derived.by(() => {
		const groups: { heading: string; events: ActivityEvent[] }[] = [];

		for (const event of events) {
			const heading = formatDayHeading(event.created_at);
			const last = groups.at(-1);

			if (last?.heading === heading) last.events.push(event);
			else groups.push({ heading, events: [event] });
		}

		return groups;
	});
</script>

{#if events.length === 0}
	<p class="empty">{empty}</p>
{:else}
	{#each days as day (day.heading)}
		<div class="day">{day.heading}</div>

		<ol>
			{#each day.events as event (event.id)}
				{@const said = describe(event)}
				<li>
					<span class="level">
						<Tag tone={said.tone} dot strong>{said.label}</Tag>
					</span>

					<div class="content">
						<p class="primary">
							<b>{said.actor}</b>
							{said.verb}
							{#if said.subject}<span class:name={said.named}>{said.subject}</span>{/if}
							{said.after}
						</p>
						<div class="secondary">
							{#if event.ip}<Tag small>{event.ip}</Tag>{/if}
							{#if said.detail}
								<Tag small tone={said.tone === 'danger' ? 'danger' : 'neutral'}>{said.detail}</Tag>
							{/if}
						</div>
					</div>

					<time datetime={event.created_at} title={formatDateTime(event.created_at)}>
						{formatTime(event.created_at)}
					</time>
				</li>
			{/each}
		</ol>
	{/each}
{/if}

<style>
	.empty {
		margin: 0;
		padding: var(--space-5) var(--space-4);
		color: var(--color-text-hint);
		text-align: center;
	}

	.day {
		padding: 5px var(--space-4);
		border-bottom: 1px solid var(--color-secondary-alt);
		background: var(--color-surface-alt);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.day:not(:first-child) {
		border-top: 1px solid var(--color-secondary-alt);
	}

	ol {
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		min-height: 54px;
		padding: 10px var(--space-4);
		transition: background var(--speed);
	}

	li + li {
		box-shadow: 0 -1px 0 0 var(--color-secondary);
	}

	li:hover {
		background: var(--color-surface-alt);
	}

	.level {
		display: flex;
		flex-shrink: 0;
		width: 9.5rem;
		padding-top: 1px;
	}

	.content {
		flex: 1;
		min-width: 0;
	}

	.primary {
		margin: 0;
		color: var(--color-text-hint);
		line-height: 1.5;
		overflow-wrap: anywhere;
	}

	.primary b,
	.name {
		color: var(--color-text);
		font-weight: 600;
	}

	.secondary {
		display: flex;
		flex-wrap: wrap;
		gap: 5px;
		margin-top: 5px;
	}

	.secondary:empty {
		display: none;
	}

	time {
		flex-shrink: 0;
		padding-top: 3px;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		white-space: nowrap;
	}

	@media (max-width: 48rem) {
		li {
			flex-wrap: wrap;
			gap: 5px var(--space-2);
		}

		.level {
			width: auto;
		}

		time {
			margin-left: auto;
		}

		.content {
			flex-basis: 100%;
			order: 3;
		}
	}
</style>
