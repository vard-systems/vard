<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { account, type Session } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { describeDevice, formatDate, timeAgo } from '#lib/utils/format.js';
	import { Badge, Button, Note, Panel } from '../ui/index.js';

	type Props = {
		sessions: Session[];
		timezone: string;
	};

	let { sessions, timezone }: Props = $props();

	const t = useTranslator();

	let ending = $state<string | null>(null);

	const others = $derived(sessions.filter((session) => !session.current).length);

	async function end(id: string) {
		ending = id;

		try {
			await account.endSession(id);
			notify.success(t('security.session_ended'));
			await refreshAll();
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			ending = null;
		}
	}
</script>

<Panel title={t('security.sessions_title')} description={t('security.sessions_description')}>
	{#snippet aside()}
		<Badge size="md">{sessions.length}</Badge>
	{/snippet}

	<ul>
		{#each sessions as session (session.id)}
			<li>
				<span class="device"><i class="ri-computer-line" aria-hidden="true"></i></span>
				<div class="details">
					<strong>
						{describeDevice(session.user_agent, t)}
						{#if session.current}
							<Badge tone="success">{t('security.this_device')}</Badge>
						{/if}
					</strong>
					<span>
						{t('security.session_line', {
							ip: session.ip || t('security.unknown_address'),
							when: timeAgo(session.signed_in_at, t),
							until: formatDate(session.expires_at, t, timezone)
						})}
					</span>
				</div>
				{#if !session.current}
					<Button
						variant="danger"
						size="sm"
						loading={ending === session.id}
						onclick={() => end(session.id)}
					>
						{t('action.sign_out')}
					</Button>
				{/if}
			</li>
		{/each}
	</ul>

	{#if others === 0}
		<Note size="sm">{t('security.no_other_sessions')}</Note>
	{/if}
</Panel>

<style>
	ul {
		display: flex;
		flex-direction: column;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--color-border);
	}

	li:first-child {
		padding-top: 0;
	}

	li:last-child {
		padding-bottom: 0;
		border-bottom: 0;
	}

	.device {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		width: 40px;
		height: 40px;
		border-radius: var(--radius-md);
		background: var(--color-input);
		color: var(--color-text-hint);
	}

	.details {
		display: flex;
		flex: 1;
		flex-direction: column;
		min-width: 0;
	}

	.details strong {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
	}

	.details > span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}
</style>
