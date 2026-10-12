<script lang="ts">
	import { messageOf, useTranslator } from '$lib/i18n';
	import { notify } from '$lib/toast.svelte';
	import { invalidateAll } from '$app/navigation';
	import { account } from '$lib/api';
	import { Button, Icon, Panel, PasswordField } from '$lib/components';
	import { describeDevice, formatDate, timeAgo } from '$lib/utils/format';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const minLength = 8;

	let current = $state('');
	let next = $state('');
	let confirm = $state('');
	let changing = $state(false);

	const mismatch = $derived(confirm !== '' && confirm !== next);
	const canChange = $derived(
		current !== '' && next.length >= minLength && confirm === next && !changing
	);

	async function changePassword(event: SubmitEvent) {
		event.preventDefault();
		if (!canChange) return;

		changing = true;

		try {
			await account.changePassword({ current_password: current, new_password: next });
			current = next = confirm = '';
			notify.success(t('security.password_changed'));
			await invalidateAll();
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			changing = false;
		}
	}

	let ending = $state<string | null>(null);

	async function endSession(id: string) {
		ending = id;

		try {
			await account.endSession(id);
			notify.success(t('security.session_ended'));
			await invalidateAll();
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			ending = null;
		}
	}

	const others = $derived(data.sessions.filter((session) => !session.current).length);
</script>

<svelte:head>
	<title>{t('security.head')} · {t('account.nav')}</title>
</svelte:head>

<div class="page">
	<div class="heading">
		<h1>{t('security.title')}</h1>
		<p>{t('security.description')}</p>
	</div>

	<form onsubmit={changePassword}>
		<Panel title={t('security.password_title')} description={t('security.password_description')}>
			<PasswordField label={t('field.current_password')} bind:value={current} disabled={changing} />
			<div class="pair">
				<PasswordField
					label={t('field.new_password')}
					bind:value={next}
					autocomplete="new-password"
					disabled={changing}
					hint={t('field.password_hint', { count: minLength })}
				/>
				<PasswordField
					label={t('field.confirm_new_password')}
					bind:value={confirm}
					autocomplete="new-password"
					disabled={changing}
					error={mismatch ? t('field.password_mismatch') : undefined}
				/>
			</div>

			{#snippet footer()}
				<Button type="submit" loading={changing} disabled={!canChange}>
					{t('security.password_title')}
				</Button>
			{/snippet}
		</Panel>
	</form>

	<Panel title={t('security.sessions_title')} description={t('security.sessions_description')}>
		{#snippet aside()}
			<span class="count">{data.sessions.length}</span>
		{/snippet}

		<ul class="sessions">
			{#each data.sessions as session (session.id)}
				<li>
					<span class="device"><Icon name="device" /></span>
					<div class="details">
						<strong>
							{describeDevice(session.user_agent, t)}
							{#if session.current}<span class="current">{t('security.this_device')}</span>{/if}
						</strong>
						<span>
							{t('security.session_line', {
								ip: session.ip || t('security.unknown_address'),
								when: timeAgo(session.signed_in_at, t),
								until: formatDate(session.expires_at, t, data.timezone)
							})}
						</span>
					</div>
					{#if !session.current}
						<Button
							variant="danger"
							size="sm"
							loading={ending === session.id}
							onclick={() => endSession(session.id)}>{t('action.sign_out')}</Button
						>
					{/if}
				</li>
			{/each}
		</ul>

		{#if others === 0}
			<p class="empty">{t('security.no_other_sessions')}</p>
		{/if}
	</Panel>
</div>

<style>
	.page {
		display: flex;
		flex-direction: column;
		gap: var(--space-5);
	}

	h1 {
		font-size: var(--text-2xl);
	}

	.heading p {
		margin-top: var(--space-1);
		color: var(--color-text-hint);
	}

	.pair {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-4);
	}

	.count {
		padding: 2px 10px;
		border-radius: 999px;
		background: var(--color-input);
		color: var(--color-text-hint);
		font-weight: 600;
	}

	.sessions {
		display: flex;
		flex-direction: column;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.sessions li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--color-border);
	}

	.sessions li:first-child {
		padding-top: 0;
	}

	.sessions li:last-child {
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

	.details span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}

	.current {
		padding: 1px 8px;
		border-radius: 999px;
		background: color-mix(in srgb, var(--color-success), transparent 86%);
		color: var(--color-success) !important;
		font-size: var(--text-xs) !important;
		font-weight: 600;
	}

	.empty {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	@media (max-width: 36rem) {
		.pair {
			grid-template-columns: 1fr;
		}
	}
</style>
