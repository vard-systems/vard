<script lang="ts">
	import { onMount } from 'svelte';
	import { browserSupportsWebAuthn, startAuthentication } from '@simplewebauthn/browser';
	import { signIn, type SignedIn } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { Button } from '../ui/index.js';

	type Props = {
		request: string | null;
		remember: boolean;
		disabled?: boolean;
		onsignedin: (result: SignedIn) => unknown;
		onerror: (message: string) => void;
	};

	let { request, remember, disabled = false, onsignedin, onerror }: Props = $props();

	const t = useTranslator();

	// Known only in the browser: the page is rendered on the server first.
	let supported = $state(false);
	let signingIn = $state(false);

	onMount(() => {
		supported = browserSupportsWebAuthn();
	});

	async function signInWithPasskey() {
		signingIn = true;

		try {
			const offer = await signIn.beginPasskey({ request: request ?? '', remember });

			let response;
			try {
				response = await startAuthentication({ optionsJSON: offer.passkey });
			} catch {
				// Closed, timed out, or no passkey for this site on the device.
				onerror(t('login.passkey_cancelled'));
				signingIn = false;
				return;
			}

			await onsignedin(await signIn.finishPasskey({ handle: offer.handle, response }));
		} catch (err) {
			onerror(messageOf(err, t));
			signingIn = false;
		}
	}
</script>

{#if supported}
	<Button
		block
		variant="secondary"
		loading={signingIn}
		disabled={disabled || signingIn}
		onclick={signInWithPasskey}
	>
		<i class="ri-fingerprint-line" aria-hidden="true"></i>
		{t('login.passkey')}
	</Button>
{/if}
