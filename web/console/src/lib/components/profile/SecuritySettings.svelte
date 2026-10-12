<script lang="ts">
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { RiComputerLine, RiShieldKeyholeLine } from 'svelte-remixicon';
	import { adminApi, messageOf, mfaApi, type AdminSession } from '$lib/api';
	import RecoveryCodes from '$lib/components/mfa/RecoveryCodes.svelte';
	import TotpSetup from '$lib/components/mfa/TotpSetup.svelte';
	import SessionList from '$lib/components/profile/SessionList.svelte';
	import {
		Alert,
		Button,
		Input,
		List,
		ListItem,
		Panel,
		Tag,
		Thumb,
		ConfirmDialog,
		notify
	} from '$lib/components/ui';
	import { keys, mfaStatusOptions, profileSessionsOptions } from '$lib/query';
	import { formatRelative } from '$lib/utils/format';

	const queryClient = useQueryClient();

	const status = createQuery(() => mfaStatusOptions());
	const ownSessions = createQuery(() => profileSessionsOptions());

	const mfa = $derived(status.data ?? null);
	const sessions = $derived(ownSessions.data ?? []);
	const loading = $derived(status.isPending || ownSessions.isPending);
	const activeSessions = $derived(sessions.filter((session) => session.active).length);

	/** Read out of the status here rather than inside the rows: a snippet is
	    a closure of its own, so narrowing `mfa` above it does not reach in. */
	const codesLeft = $derived(mfa?.recovery_codes_left ?? 0);

	type Mode = 'reading' | 'enrolling' | 'replacing' | 'codes';

	let mode = $state<Mode>('reading');

	let asking = $state<'disable' | 'codes' | null>(null);
	let code = $state('');
	let busy = $state(false);
	let newCodes = $state<string[]>([]);

	const error = $derived(
		status.error || ownSessions.error
			? messageOf(status.error ?? ownSessions.error, 'Could not read your account')
			: ''
	);

	let ending = $state<string | null>(null);
	let confirmingOthers = $state(false);
	let endingOthers = $state(false);

	const otherSessions = $derived(sessions.filter((session) => session.active && !session.current));

	async function endSession(session: AdminSession) {
		ending = session.id;

		try {
			await adminApi.endSession(session.id);
			notify.success('Session signed out');
			await queryClient.invalidateQueries({ queryKey: keys.profile.sessions });
		} catch (err) {
			notify.error(err, 'Could not sign that session out');
		} finally {
			ending = null;
		}
	}

	async function endOthers() {
		endingOthers = true;

		try {
			const { ended } = await adminApi.endOtherSessions();
			notify.success(
				ended === 1 ? 'One other session signed out' : `${ended} other sessions signed out`
			);
			confirmingOthers = false;
			await queryClient.invalidateQueries({ queryKey: keys.profile.sessions });
		} catch (err) {
			confirmingOthers = false;
			notify.error(err, 'Could not sign the other sessions out');
		} finally {
			endingOthers = false;
		}
	}

	function backToReading() {
		mode = 'reading';
		asking = null;
		code = '';
		newCodes = [];
		void queryClient.invalidateQueries({ queryKey: keys.profile.all });
	}

	async function disable() {
		busy = true;

		try {
			await mfaApi.disable(code.trim());
			notify.success('Two-factor sign-in turned off');
			backToReading();
		} catch (err) {
			notify.error(err, 'Could not turn two-factor sign-in off');
		} finally {
			busy = false;
		}
	}

	async function regenerate() {
		busy = true;

		try {
			const { recovery_codes } = await mfaApi.recoveryCodes(code.trim());
			notify.success('New recovery codes made', 'Save them now: the old ones no longer work.');
			newCodes = recovery_codes;
			mode = 'codes';
			asking = null;
			code = '';
		} catch (err) {
			notify.error(err, 'Could not make new recovery codes');
		} finally {
			busy = false;
		}
	}
</script>

<div class="sections">
	{#if mode === 'enrolling' || mode === 'replacing'}
		<Panel
			title={mode === 'replacing' ? 'Replace your authenticator' : 'Set up an authenticator'}
			icon={RiShieldKeyholeLine}
		>
			<TotpSetup
				replacing={mode === 'replacing'}
				onDone={backToReading}
				onCancel={() => (mode = 'reading')}
			/>
		</Panel>
	{:else if mode === 'codes'}
		<Panel title="New recovery codes" icon={RiShieldKeyholeLine}>
			<RecoveryCodes codes={newCodes} onDone={backToReading} />
		</Panel>
	{:else}
		{#if error}<Alert>{error}</Alert>{/if}

		<Panel title="Two-factor sign-in" icon={RiShieldKeyholeLine} flush>
			{#snippet meta()}
				{#if loading}
					<Tag>Loading</Tag>
				{:else if mfa?.enabled}
					<Tag tone="success" dot strong>On</Tag>
				{:else if mfa?.required}
					<Tag tone="danger" dot strong>Required</Tag>
				{:else if mfa}
					<Tag dot>Off</Tag>
				{:else}
					<Tag>Unknown</Tag>
				{/if}
			{/snippet}

			{#if !mfa}
				<p class="quiet padded">{loading ? 'Reading your account…' : 'Not available.'}</p>
			{:else if !mfa.enabled}
				<div class="prose">
					<p>
						An authenticator app asks for a six-digit code when you sign in, so a password on its
						own is not enough to get in as you.
					</p>
					<Button size="sm" onclick={() => (mode = 'enrolling')}>Set one up</Button>
				</div>
			{:else}
				<List label="Two-factor">
					<ListItem
						title="Authenticator"
						description={mfa.confirmed_at ? `Set up ${formatRelative(mfa.confirmed_at)}` : 'Set up'}
					>
						{#snippet lead()}<Thumb icon={RiShieldKeyholeLine} />{/snippet}
						{#snippet end()}
							<Button size="sm" variant="subtle" onclick={() => (mode = 'replacing')}>
								Replace
							</Button>
						{/snippet}
					</ListItem>

					<ListItem
						title="Recovery codes"
						description="Each one signs you in once if you lose your phone."
					>
						{#snippet end()}
							<Tag small tone={codesLeft > 2 ? 'neutral' : 'warning'}>
								{codesLeft} left
							</Tag>
							<Button size="sm" variant="subtle" onclick={() => (asking = 'codes')}>New</Button>
						{/snippet}
					</ListItem>

					{#if !mfa.required}
						<ListItem title="Turn it off" description="A password alone will sign you in again.">
							{#snippet end()}
								<Button
									size="sm"
									variant="subtle"
									colorPalette="danger"
									onclick={() => (asking = 'disable')}
								>
									Turn off
								</Button>
							{/snippet}
						</ListItem>
					{/if}
				</List>

				{#if asking}
					<div class="asking">
						<Input
							label={asking === 'disable'
								? 'Code from your authenticator, to turn it off'
								: 'Code from your authenticator, for new codes'}
							bind:value={code}
							inputmode="numeric"
							autocomplete="one-time-code"
							maxlength={16}
							disabled={busy}
						/>

						<div class="buttons">
							<Button
								size="sm"
								variant="subtle"
								disabled={busy}
								onclick={() => {
									asking = null;
									code = '';
								}}
							>
								Cancel
							</Button>
							<Button
								size="sm"
								colorPalette={asking === 'disable' ? 'danger' : 'neutral'}
								loading={busy}
								disabled={busy || code.trim() === ''}
								onclick={asking === 'disable' ? disable : regenerate}
							>
								{asking === 'disable' ? 'Turn off' : 'Make new codes'}
							</Button>
						</div>
					</div>
				{/if}
			{/if}
		</Panel>

		<Panel title="Sessions" icon={RiComputerLine} flush>
			{#snippet meta()}
				{#if loading}
					<Tag>Loading</Tag>
				{:else}
					<Tag tone={activeSessions > 0 ? 'success' : 'neutral'} dot small>
						{activeSessions} active
					</Tag>
				{/if}
			{/snippet}

			{#if loading && sessions.length === 0}
				<p class="quiet padded">Reading your sessions…</p>
			{:else}
				<SessionList {sessions} onEnd={endSession} {ending} />

				{#if otherSessions.length > 0}
					<div class="asking">
						<div class="buttons">
							<Button
								size="sm"
								variant="subtle"
								colorPalette="danger"
								onclick={() => (confirmingOthers = true)}
							>
								Sign out other sessions
							</Button>
						</div>
					</div>
				{/if}
			{/if}
		</Panel>
	{/if}
</div>

<ConfirmDialog
	bind:open={confirmingOthers}
	title={`Sign out of ${otherSessions.length} other ${otherSessions.length === 1 ? 'session' : 'sessions'}?`}
	description="Every other browser signed in as you is signed out at once. This one stays signed in."
	tone="warning"
	confirmLabel="Sign out everywhere else"
	busy={endingOthers}
	onConfirm={endOthers}
/>

<style>
	.sections {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.quiet {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.padded {
		padding: var(--space-4);
	}

	.prose {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: var(--space-3);
		padding: var(--space-4);
	}

	.prose p {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.55;
	}

	.asking {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		padding: var(--space-4);
		border-top: 1px solid var(--color-border);
		background: var(--color-surface-alt);
	}

	.buttons {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
