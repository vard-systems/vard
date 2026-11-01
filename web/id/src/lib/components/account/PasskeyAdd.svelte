<script lang="ts" generics="Added">
	import { browserSupportsWebAuthn, startRegistration } from '@simplewebauthn/browser';
	import type { PublicKeyCredentialCreationOptionsJSON } from '@simplewebauthn/browser';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { Button, Form, TextField } from '../ui/index.js';

	type Props = {
		/** Starts the ceremony: the account page's call, or the waiting sign-in's. */
		begin: () => Promise<{ passkey: PublicKeyCredentialCreationOptionsJSON }>;
		/** Keeps the passkey the device made, under its name. */
		finish: (body: { label: string; response: unknown }) => Promise<Added>;
		/** Called with what finishing answered. */
		onadded: (added: Added) => unknown;
		oncancel: () => void;
	};

	let { begin, finish, onadded, oncancel }: Props = $props();

	const t = useTranslator();

	let label = $state('');
	let adding = $state(false);

	async function add() {
		if (!browserSupportsWebAuthn()) {
			notify.error(t('security.passkey_unsupported'));
			return;
		}

		adding = true;

		try {
			const { passkey } = await begin();

			let response;
			try {
				response = await startRegistration({ optionsJSON: passkey });
			} catch {
				// Closed, timed out, or refused by the device: nothing was made.
				notify.error(t('security.passkey_cancelled'));
				adding = false;
				return;
			}

			const added = await finish({ label: label.trim(), response });
			notify.success(t('security.passkey_added'));
			await onadded(added);
		} catch (err) {
			notify.error(messageOf(err, t));
			adding = false;
		}
	}
</script>

<Form onsubmit={add}>
	<TextField
		label={t('security.passkey_name')}
		bind:value={label}
		placeholder={t('security.passkey_name_placeholder')}
		maxlength={100}
		disabled={adding}
	/>
	<div class="actions">
		<Button variant="ghost" size="sm" onclick={oncancel} disabled={adding}>
			{t('action.cancel')}
		</Button>
		<Button type="submit" size="sm" loading={adding}>
			<i class="ri-fingerprint-line" aria-hidden="true"></i>
			{t('security.passkey_add')}
		</Button>
	</div>
</Form>

<style>
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
