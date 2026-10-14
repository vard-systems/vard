<script lang="ts">
	import {
		RiComputerLine,
		RiGlobalLine,
		RiLoginBoxLine,
		RiLogoutBoxRLine,
		RiTimeLine,
		RiUserLine
	} from 'svelte-remixicon';
	import type { UserSessionRecord } from '#lib/api/index.js';
	import { Button, DataTable, Icon, type Column } from '#lib/components/ui/index.js';
	import { describeUserAgent, formatDateTime, formatRelative } from '#lib/utils/format.js';

	type Props = {
		sessions: UserSessionRecord[];
		empty: string;
		onEnd?: (session: UserSessionRecord) => void;
		onSignOutUser?: (session: UserSessionRecord) => void;
		onUser: (session: UserSessionRecord) => void;
		ending?: string;
	};

	let { sessions, empty, onEnd, onSignOutUser, onUser, ending }: Props = $props();

	const columns: Column[] = $derived([
		{ key: 'user', label: 'User', icon: RiUserLine, min: '16rem' },
		{ key: 'device', label: 'Device', icon: RiComputerLine, min: '12rem' },
		{ key: 'ip', label: 'IP address', icon: RiGlobalLine, min: '8rem' },
		{ key: 'signed_in', label: 'Signed in', icon: RiLoginBoxLine, min: '9rem' },
		{ key: 'expires', label: 'Expires', icon: RiTimeLine, min: '11rem' },
		...(onEnd ? [{ key: 'actions', label: '', min: '16rem', align: 'end' as const }] : [])
	]);
</script>

<DataTable {columns} rows={sessions} {empty}>
	{#snippet row(session)}
		<td>
			<button
				type="button"
				class="user"
				title={`Only ${session.user.email}`}
				onclick={() => onUser(session)}
			>
				<span class="email">{session.user.email}</span>
				{#if session.user.name}<span class="name">{session.user.name}</span>{/if}
			</button>
		</td>

		<td>{describeUserAgent(session.user_agent)}</td>

		<td><span class="mono">{session.ip || 'unknown'}</span></td>

		<td>
			<time datetime={session.signed_in_at} title={formatDateTime(session.signed_in_at)}>
				{formatRelative(session.signed_in_at)}
			</time>
		</td>

		<td><span class="mono">{formatDateTime(session.expires_at)}</span></td>

		{#if onEnd && onSignOutUser}
			<td class="actions">
				<Button
					variant="subtle"
					size="sm"
					title="Sign this session out"
					loading={ending === session.id}
					onclick={() => onEnd(session)}
				>
					Sign out
				</Button>
				<Button
					variant="subtle"
					colorPalette="danger"
					size="sm"
					title={`End every session ${session.user.email} has and revoke their applications' tokens`}
					onclick={() => onSignOutUser(session)}
				>
					<Icon icon={RiLogoutBoxRLine} />
					Sign out everywhere
				</Button>
			</td>
		{/if}
	{/snippet}
</DataTable>

<style>
	.user {
		display: inline-flex;
		flex-direction: column;
		align-items: flex-start;
		padding: 0;
		border: none;
		background: none;
		color: var(--color-text);
		font: inherit;
		text-align: left;
		cursor: pointer;
	}

	.user:hover .email {
		text-decoration: underline;
	}

	.name {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.mono {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.actions {
		text-align: right;
		white-space: nowrap;
	}

	.actions :global(button + button) {
		margin-left: var(--space-1);
	}
</style>
