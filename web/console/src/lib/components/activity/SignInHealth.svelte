<script lang="ts">
	import { RiErrorWarningLine, RiLockLine, RiShieldCheckLine } from 'svelte-remixicon';
	import type { Overview } from '#lib/api/index.js';
	import { Alert, Icon, List, ListItem, Tag } from '#lib/components/ui/index.js';

	type Props = {
		signIns: Overview['sign_ins'];
		locked: number;
		hrefs?: { succeeded?: string; failed?: string; blocked?: string };
	};

	let { signIns, locked, hrefs = {} }: Props = $props();

	const attempts = $derived(signIns.succeeded + signIns.failed + signIns.blocked);
	const rate = $derived(attempts === 0 ? null : Math.round((signIns.succeeded / attempts) * 100));

	const share = (value: number) => (attempts === 0 ? 0 : (value / attempts) * 100);

	const rows = $derived([
		{
			key: 'success' as const,
			label: 'Successful',
			value: signIns.succeeded,
			icon: RiShieldCheckLine,
			href: hrefs.succeeded
		},
		{
			key: 'danger' as const,
			label: 'Wrong password',
			value: signIns.failed,
			icon: RiErrorWarningLine,
			href: hrefs.failed
		},
		{
			key: 'warning' as const,
			label: 'Blocked account',
			value: signIns.blocked,
			icon: RiLockLine,
			href: hrefs.blocked
		}
	]);
</script>

<div class="summary">
	<div>
		<span class="hint">Success rate</span>
		<strong>{rate === null ? '—' : `${rate}%`}</strong>
	</div>
	<Tag>{attempts.toLocaleString()} {attempts === 1 ? 'attempt' : 'attempts'}</Tag>
</div>

<div class="meter" aria-hidden="true">
	{#if attempts > 0}
		{#each rows as row (row.key)}
			{#if row.value > 0}
				<span class={row.key} style:width="{share(row.value)}%"></span>
			{/if}
		{/each}
	{/if}
</div>

<List bordered label="Sign-in attempts">
	{#each rows as row (row.key)}
		<ListItem href={row.value > 0 ? row.href : undefined}>
			{#snippet lead()}<span class="icon"><Icon icon={row.icon} /></span>{/snippet}
			<span class="name">{row.label}</span>
			{#snippet end()}
				<Tag tone={row.value > 0 ? row.key : 'neutral'} dot strong>{row.value.toLocaleString()}</Tag
				>
			{/snippet}
		</ListItem>
	{/each}
</List>

{#if locked > 0}
	<div class="alert">
		<Alert tone="warning">
			<b>{locked} {locked === 1 ? 'administrator is' : 'administrators are'} locked out</b> after too
			many wrong passwords. The lock lifts on its own, or with a new password.
		</Alert>
	</div>
{/if}

<style>
	.summary {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: var(--space-2);
	}

	.summary div {
		display: flex;
		flex-direction: column;
	}

	.hint {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	strong {
		font-size: 1.714rem;
		font-weight: 600;
		line-height: 1.3;
		font-variant-numeric: tabular-nums;
	}

	.meter {
		display: flex;
		gap: 2px;
		height: 6px;
		margin: var(--space-2) 0 var(--space-3);
		overflow: hidden;
		border-radius: var(--radius-sm);
		background: var(--color-secondary);
	}

	.meter .success {
		background: var(--color-success);
	}

	.meter .danger {
		background: var(--color-danger);
	}

	.meter .warning {
		background: var(--color-warning);
	}

	.icon {
		display: inline-flex;
		color: var(--color-text-hint);
	}

	.name {
		font-size: var(--text-sm);
	}

	.alert {
		margin-top: var(--space-3);
		font-size: var(--text-sm);
	}
</style>
