<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { account, type TwoStep } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { Badge, Button, ItemList, ItemRow, Note, Panel } from '../ui/index.js';
	import AuthenticatorSetup from './AuthenticatorSetup.svelte';
	import FactorRow from './FactorRow.svelte';
	import PasskeyAdd from './PasskeyAdd.svelte';
	import PresenceCheck from './PresenceCheck.svelte';
	import RecoveryCodeSheet from './RecoveryCodeSheet.svelte';

	type Props = { twoStep: TwoStep; timezone: string };

	let { twoStep, timezone }: Props = $props();

	const t = useTranslator();

	/** What is open: one thing at a time, so the panel never grows two forms. */
	let open = $state<'authenticator' | 'passkey' | 'codes' | null>(null);
	/** Adding a factor is confirmed first, with the password when there is one. */
	let confirmed = $state(false);
	let password = $state('');

	function start(what: 'authenticator' | 'passkey') {
		open = what;
		password = '';
		confirmed = !twoStep.asks_password;
	}

	function close() {
		open = null;
		confirmed = false;
		password = '';
	}
	/** Recovery codes just made, shown until the reader says they kept them. */
	let shown = $state<string[]>([]);
	let replacing = $state(false);

	async function added(codes: string[]) {
		close();
		shown = codes;
		await refreshAll();
	}

	async function replaceCodes(password: string) {
		replacing = true;

		try {
			const { recovery_codes } = await account.newRecoveryCodes(password);
			shown = recovery_codes ?? [];
			open = null;
			await refreshAll();
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			replacing = false;
		}
	}
</script>

{#snippet confirm()}
	<PresenceCheck
		asksPassword
		confirm={t('security.continue')}
		onconfirm={(typed) => ((password = typed), (confirmed = true))}
		oncancel={close}
	/>
{/snippet}

{#snippet authenticatorSetup()}
	{#if confirmed}
		<AuthenticatorSetup {password} onadded={added} oncancel={close} />
	{:else}
		{@render confirm()}
	{/if}
{/snippet}

{#snippet passkeyAdd()}
	{#if confirmed}
		<PasskeyAdd {password} onadded={added} oncancel={close} />
	{:else}
		{@render confirm()}
	{/if}
{/snippet}

{#snippet codesCheck()}
	<PresenceCheck
		asksPassword={twoStep.asks_password}
		confirm={t('security.recovery_new')}
		busy={replacing}
		onconfirm={replaceCodes}
		oncancel={() => (open = null)}
	/>
{/snippet}

<Panel title={t('security.two_step_title')} description={t('security.two_step_description')}>
	{#snippet aside()}
		<Badge
			size="md"
			tone={twoStep.enabled ? 'success' : 'neutral'}
			icon={twoStep.enabled ? 'ri-shield-check-line' : undefined}
		>
			{twoStep.enabled ? t('security.two_step_on') : t('security.two_step_off')}
		</Badge>
	{/snippet}

	{#if shown.length > 0}
		<RecoveryCodeSheet codes={shown} ondone={() => (shown = [])} />
	{/if}

	{#if !twoStep.offered && !twoStep.enabled}
		<Note size="sm">{t('security.two_step_not_offered')}</Note>
	{:else}
		<ItemList>
			{#if twoStep.authenticator}
				<FactorRow
					factor={twoStep.authenticator}
					icon="ri-smartphone-line"
					name={t('security.authenticator_title')}
					asksPassword={twoStep.asks_password}
					{timezone}
				/>
			{:else if twoStep.offered}
				<ItemRow
					icon="ri-smartphone-line"
					detail={t('security.authenticator_description')}
					below={open === 'authenticator' ? authenticatorSetup : undefined}
				>
					{#snippet title()}{t('security.authenticator_title')}{/snippet}
					{#snippet actions()}
						{#if open !== 'authenticator'}
							<Button variant="secondary" size="sm" onclick={() => start('authenticator')}>
								{t('security.authenticator_set_up')}
							</Button>
						{/if}
					{/snippet}
				</ItemRow>
			{/if}

			{#each twoStep.passkeys as passkey (passkey.id)}
				<FactorRow
					factor={passkey}
					icon="ri-fingerprint-line"
					name={passkey.label || t('security.passkey_unnamed')}
					asksPassword={twoStep.asks_password}
					{timezone}
				/>
			{/each}

			{#if twoStep.offered}
				<ItemRow
					icon="ri-fingerprint-line"
					detail={t('security.passkeys_description')}
					below={open === 'passkey' ? passkeyAdd : undefined}
				>
					{#snippet title()}{t('security.passkeys_title')}{/snippet}
					{#snippet actions()}
						{#if open !== 'passkey'}
							<Button variant="secondary" size="sm" onclick={() => start('passkey')}>
								{t('security.passkey_add')}
							</Button>
						{/if}
					{/snippet}
				</ItemRow>
			{/if}

			{#if twoStep.enabled}
				<ItemRow
					icon="ri-key-line"
					detail={t('security.recovery_left', { count: twoStep.recovery_codes_left })}
					below={open === 'codes' ? codesCheck : undefined}
				>
					{#snippet title()}{t('security.recovery_title')}{/snippet}
					{#snippet actions()}
						{#if open !== 'codes'}
							<Button variant="secondary" size="sm" onclick={() => (open = 'codes')}>
								{t('security.recovery_new')}
							</Button>
						{/if}
					{/snippet}
				</ItemRow>
			{/if}
		</ItemList>
	{/if}
</Panel>
