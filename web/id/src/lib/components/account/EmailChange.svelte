<script lang="ts">
	import { account } from '#lib/api/index.js';
	import { LIMITS } from '#lib/constants.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { Button, PasswordField, TextField } from '../ui/index.js';

	const t = useTranslator();

	let open = $state(false);
	let email = $state('');
	/** The address and a password reset are the whole account, so the
	    server asks for the password before it sends the link. */
	let password = $state('');
	let sending = $state(false);

	const canSend = $derived(email.trim() !== '' && password !== '' && !sending);

	function start() {
		email = '';
		password = '';
		open = true;
	}

	// Inside the profile's form, which a form cannot be: sent by its button.
	async function send() {
		if (!canSend) return;

		const address = email.trim();
		sending = true;

		try {
			await account.changeEmail({ email: address, current_password: password });
			notify.success(t('account.email_change_sent', { email: address }));
			open = false;
			password = '';
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			sending = false;
		}
	}
</script>

{#if open}
	<TextField
		label={t('account.email_change_title')}
		bind:value={email}
		type="email"
		autocomplete="email"
		autocapitalize="none"
		spellcheck={false}
		maxlength={LIMITS.email}
		disabled={sending}
	/>
	<PasswordField label={t('field.current_password')} bind:value={password} disabled={sending} />

	<div class="actions">
		<Button variant="secondary" disabled={sending} onclick={() => (open = false)}>
			{t('account.email_change_cancel')}
		</Button>
		<Button loading={sending} disabled={!canSend} onclick={send}>
			{t('account.email_change_submit')}
		</Button>
	</div>
{:else}
	<div class="actions">
		<Button variant="secondary" onclick={start}>{t('account.email_change')}</Button>
	</div>
{/if}

<style>
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
