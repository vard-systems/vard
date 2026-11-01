<script lang="ts" generics="Added">
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';
	import type { AuthenticatorSecret } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { Button, Form, TextField } from '../ui/index.js';

	type Props = {
		/** Starts the setup: the account page's call, or the waiting sign-in's. */
		begin: () => Promise<{ authenticator: AuthenticatorSecret }>;
		/** Confirms it with a code from the app. */
		confirm: (code: string) => Promise<Added>;
		/** Called with what confirming answered. */
		onadded: (added: Added) => unknown;
		oncancel: () => void;
	};

	let { begin, confirm: send, onadded, oncancel }: Props = $props();

	const t = useTranslator();

	let secret = $state('');
	let qr = $state('');
	let code = $state('');
	let starting = $state(true);
	let confirming = $state(false);

	async function start() {
		try {
			const { authenticator } = await begin();
			secret = authenticator.secret;
			qr = await QRCode.toString(authenticator.uri, { type: 'svg', margin: 1, width: 176 });
		} catch (err) {
			notify.error(messageOf(err, t));
			oncancel();
		} finally {
			starting = false;
		}
	}

	async function confirm() {
		if (code.trim() === '') return;

		confirming = true;

		try {
			const added = await send(code);
			notify.success(t('security.authenticator_added'));
			await onadded(added);
		} catch (err) {
			notify.error(messageOf(err, t));
			confirming = false;
		}
	}

	onMount(() => void start());
</script>

<Form onsubmit={confirm}>
	<p class="lead">{t('security.authenticator_scan')}</p>

	<div class="setup">
		<div class="qr" aria-busy={starting}>
			<!-- eslint-disable-next-line svelte/no-at-html-tags -- an SVG drawn here from the server's own URI -->
			{#if qr}{@html qr}{/if}
		</div>

		<div class="fields">
			<TextField
				label={t('security.authenticator_key')}
				value={secret}
				readonly
				spellcheck={false}
			/>
			<TextField
				label={t('security.authenticator_code')}
				bind:value={code}
				inputmode="numeric"
				autocomplete="one-time-code"
				maxlength={8}
				disabled={confirming}
			/>
		</div>
	</div>

	<div class="actions">
		<Button variant="ghost" size="sm" onclick={oncancel} disabled={confirming}>
			{t('action.cancel')}
		</Button>
		<Button type="submit" size="sm" loading={confirming} disabled={code.trim() === '' || starting}>
			{t('security.authenticator_confirm')}
		</Button>
	</div>
</Form>

<style>
	.lead {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.setup {
		display: grid;
		grid-template-columns: auto 1fr;
		align-items: start;
		gap: var(--space-4);
	}

	/* A QR code has to stay dark on light to scan, whatever the theme. */
	.qr {
		display: grid;
		place-items: center;
		width: 192px;
		aspect-ratio: 1;
		padding: var(--space-2);
		border-radius: var(--radius-md);
		background: #fff;
	}

	.qr :global(svg) {
		display: block;
		width: 100%;
		height: auto;
	}

	.fields {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		min-width: 0;
	}

	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}

	@media (max-width: 30rem) {
		.setup {
			grid-template-columns: 1fr;
			justify-items: center;
		}

		.fields {
			width: 100%;
		}
	}
</style>
