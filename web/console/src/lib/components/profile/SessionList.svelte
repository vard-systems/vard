<script lang="ts">
	import { RiComputerLine, RiSmartphoneLine } from 'svelte-remixicon';
	import type { AdminSession } from '#lib/api/index.js';
	import { Button, List, ListItem, Tag, Thumb } from '#lib/components/ui/index.js';
	import { describeUserAgent, formatDateTime, formatRelative } from '#lib/utils/format.js';

	type Props = {
		sessions: AdminSession[];
		limit?: number;
		bordered?: boolean;
		onEnd?: (session: AdminSession) => void;
		ending?: string | null;
	};

	let { sessions, limit, bordered = false, onEnd, ending = null }: Props = $props();

	const shown = $derived(
		[...sessions]
			.sort(
				(a, b) =>
					Number(b.current) - Number(a.current) ||
					Number(b.active) - Number(a.active) ||
					b.created_at.localeCompare(a.created_at)
			)
			.slice(0, limit)
	);

	const mobile = (agent: string) => /iPhone|iPad|Android|Mobile/.test(agent);
</script>

{#if shown.length === 0}
	<p class="empty">No sessions recorded.</p>
{:else}
	<List {bordered} label="Sessions">
		{#each shown as session (session.id)}
			<ListItem title={describeUserAgent(session.user_agent)}>
				{#snippet lead()}
					<Thumb icon={mobile(session.user_agent) ? RiSmartphoneLine : RiComputerLine} />
				{/snippet}

				<span class="meta">
					<span class="ip">{session.ip || '—'}</span>
					<time datetime={session.created_at} title={formatDateTime(session.created_at)}>
						signed in {formatRelative(session.created_at)}
					</time>
				</span>

				{#snippet end()}
					{#if session.current}
						<Tag tone="info" strong>This device</Tag>
					{:else if session.active && onEnd}
						<Button
							size="sm"
							variant="subtle"
							colorPalette="danger"
							loading={ending === session.id}
							disabled={ending !== null}
							onclick={() => onEnd(session)}
						>
							Sign out
						</Button>
					{:else}
						<Tag tone={session.active ? 'success' : 'neutral'} dot strong>
							{session.active ? 'Active' : 'Ended'}
						</Tag>
					{/if}
				{/snippet}
			</ListItem>
		{/each}
	</List>
{/if}

<style>
	.empty {
		margin: 0;
		padding: var(--space-4);
		color: var(--color-text-hint);
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		gap: 0 var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.ip {
		font-family: var(--font-mono);
	}
</style>
