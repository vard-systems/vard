<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { account, type Session } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { describeDevice, formatDate, timeAgo } from '#lib/utils/format.js';
	import { Badge, Button, ItemList, ItemRow, Note, Panel } from '../ui/index.js';

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
		<Badge>{sessions.length}</Badge>
	{/snippet}

	<ItemList>
		{#each sessions as session (session.id)}
			<ItemRow
				icon="ri-computer-line"
				detail={t('security.session_line', {
					ip: session.ip || t('security.unknown_address'),
					when: timeAgo(session.signed_in_at, t),
					until: formatDate(session.expires_at, t, timezone)
				})}
			>
				{#snippet title()}
					{describeDevice(session.user_agent, t)}
					{#if session.current}
						<Badge tone="success">{t('security.this_device')}</Badge>
					{/if}
				{/snippet}
				{#snippet actions()}
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
				{/snippet}
			</ItemRow>
		{/each}
	</ItemList>

	{#if others === 0}
		<Note>{t('security.no_other_sessions')}</Note>
	{/if}
</Panel>
