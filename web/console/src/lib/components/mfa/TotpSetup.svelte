<script lang="ts">
	import { onMount } from 'svelte';
	import QRCode from 'qrcode';
	import { ApiError, mfaApi, type MfaEnrolment } from '#lib/api/index.js';
	import { Alert, Button, CopyButton, Input } from '#lib/components/ui/index.js';
	import RecoveryCodes from './RecoveryCodes.svelte';

	type Props = {
		replacing?: boolean;
		doneLabel?: string;
		onDone: () => void;
		onCancel?: () => void;
	};

	let { replacing = false, doneLabel = 'Done', onDone, onCancel }: Props = $props();

	type Step = 'prove' | 'scan' | 'codes';

	// svelte-ignore state_referenced_locally
	let step = $state<Step>(replacing ? 'prove' : 'scan');
	let enrolment = $state<MfaEnrolment | null>(null);
	let qr = $state('');
	let currentCode = $state('');
	let code = $state('');
	let codes = $state<string[]>([]);
	let error = $state('');
	let busy = $state(false);

	function message(err: unknown) {
		return err instanceof ApiError ? err.message : 'Something went wrong';
	}

	const grouped = $derived(enrolment?.secret.match(/.{1,4}/g)?.join(' ') ?? '');

	async function begin(proof?: string) {
		busy = true;
		error = '';

		try {
			({ enrolment } = await mfaApi.begin(proof));
			qr = await QRCode.toString(enrolment.uri, { type: 'svg', margin: 1, width: 196 });
			step = 'scan';
		} catch (err) {
			error = message(err);
		} finally {
			busy = false;
		}
	}

	async function confirm(event: SubmitEvent) {
		event.preventDefault();
		if (code.trim().length < 6) return;

		busy = true;
		error = '';

		try {
			({ recovery_codes: codes } = await mfaApi.confirm(code));
			step = 'codes';
		} catch (err) {
			error = message(err);
			code = '';
		} finally {
			busy = false;
		}
	}

	onMount(() => {
		if (!replacing) begin();
	});
</script>

<div class="setup">
	{#if error}<Alert>{error}</Alert>{/if}

	{#if step === 'prove'}
		<form
			onsubmit={(event) => {
				event.preventDefault();
				begin(currentCode);
			}}
		>
			<p class="text">
				Enter a code from your current authenticator, or a recovery code, to replace it.
			</p>
			<Input
				label="Current code"
				bind:value={currentCode}
				inputmode="numeric"
				autocomplete="one-time-code"
				autocapitalize="none"
				spellcheck={false}
				disabled={busy}
			/>
			<div class="row">
				{#if onCancel}<Button variant="subtle" onclick={onCancel} disabled={busy}>Cancel</Button
					>{/if}
				<Button type="submit" loading={busy} disabled={currentCode.trim().length < 6}>
					Continue
				</Button>
			</div>
		</form>
	{:else if step === 'scan'}
		<ol class="steps">
			<li>
				<span class="text"
					>Scan this QR code with an authenticator app — Google Authenticator, 1Password, Authy,
					Microsoft Authenticator.</span
				>
				<div class="qr" aria-busy={!qr}>
					{#if qr}
						<!-- The SVG is generated here from the otpauth URI, not taken from the server. -->
						<!-- eslint-disable-next-line svelte/no-at-html-tags -->
						{@html qr}
					{/if}
				</div>
				{#if enrolment}
					<details>
						<summary>Can’t scan it? Enter the key instead</summary>
						<div class="secret">
							<code>{grouped}</code>
							<CopyButton value={enrolment.secret} label="setup key" />
						</div>
					</details>
				{/if}
			</li>
			<li>
				<form onsubmit={confirm}>
					<span class="text">Enter the six-digit code the app shows.</span>
					<Input
						label="Code"
						bind:value={code}
						inputmode="numeric"
						autocomplete="one-time-code"
						maxlength={7}
						disabled={busy || !enrolment}
					/>
					<div class="row">
						{#if onCancel}<Button variant="subtle" onclick={onCancel} disabled={busy}>Cancel</Button
							>{/if}
						<Button type="submit" loading={busy} disabled={!enrolment || code.trim().length < 6}>
							Verify and turn on
						</Button>
					</div>
				</form>
			</li>
		</ol>
	{:else}
		<RecoveryCodes {codes} {doneLabel} {onDone} />
	{/if}
</div>

<style>
	.setup,
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.steps {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		margin: 0;
		padding-left: 1.25rem;
	}

	.steps li {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.text {
		color: var(--color-text);
		line-height: 1.5;
	}

	.qr {
		display: grid;
		place-items: center;
		width: 212px;
		height: 212px;
		padding: 8px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		/* A QR code has to stay dark on light to scan, whatever the theme. */
		background: #fff;
	}

	.qr :global(svg) {
		display: block;
		width: 196px;
		height: 196px;
	}

	details summary {
		color: var(--color-text-hint);
		cursor: pointer;
		font-size: var(--text-sm);
	}

	.secret {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-top: var(--space-2);
	}

	.secret code {
		font-family: var(--font-mono);
		font-size: var(--text-base);
		overflow-wrap: anywhere;
	}

	.row {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
