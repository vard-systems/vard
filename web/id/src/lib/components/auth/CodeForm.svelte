<script lang="ts">
	import { ApiError, signIn, type CodeChallenge, type SignedIn } from '#lib/api/index.js';
	import { LIMITS } from '#lib/constants.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { Alert, Button, Form, TextField } from '../ui/index.js';

	type Props = {
		challenge: CodeChallenge;
		onsignedin: (result: SignedIn) => void;
		onexpired: (message: string) => void;
	};

	let { challenge, onsignedin, onexpired }: Props = $props();

	const t = useTranslator();

	// svelte-ignore state_referenced_locally
	let waiting = $state(challenge);

	let code = $state('');
	let error = $state('');
	let notice = $state('');
	let submitting = $state(false);
	let resending = $state(false);

	// svelte-ignore state_referenced_locally
	let countdown = $state(challenge.resend_after);

	$effect(() => {
		if (countdown <= 0) return;

		const timer = setInterval(() => {
			countdown = Math.max(countdown - 1, 0);
		}, 1000);

		return () => clearInterval(timer);
	});

	const canSubmit = $derived(code.trim() !== '' && !submitting);

	function isOver(err: unknown): boolean {
		return (
			err instanceof ApiError && (err.code === 'code_expired' || err.code === 'code_attempts_used')
		);
	}

	async function submit() {
		if (!canSubmit) return;

		error = '';
		notice = '';
		submitting = true;

		try {
			onsignedin(await signIn.code({ handle: waiting.handle, code: code.trim() }));
		} catch (err) {
			if (isOver(err)) {
				onexpired(messageOf(err, t));
				return;
			}

			error = messageOf(err, t);
			code = '';
			waiting = { ...waiting, attempts_left: Math.max(waiting.attempts_left - 1, 0) };
			submitting = false;
		}
	}

	async function resend() {
		if (resending || countdown > 0) return;

		error = '';
		notice = '';
		resending = true;

		try {
			const { code: next } = await signIn.resendCode({ handle: waiting.handle });
			waiting = next;
			countdown = next.resend_after;
			code = '';
			notice = t('login.code_resent');
		} catch (err) {
			if (isOver(err)) {
				onexpired(messageOf(err, t));
				return;
			}

			error = messageOf(err, t);
		} finally {
			resending = false;
		}
	}
</script>

<Form onsubmit={submit}>
	{#if error}<Alert>{error}</Alert>{/if}
	{#if notice}<Alert tone="info">{notice}</Alert>{/if}

	<TextField
		label={t('login.code_label')}
		bind:value={code}
		name="one-time-code"
		inputmode="numeric"
		autocomplete="one-time-code"
		autocapitalize="none"
		spellcheck={false}
		maxlength={LIMITS.code}
		disabled={submitting}
		hint={t('login.code_attempts_left', { count: waiting.attempts_left })}
	/>

	<Button type="submit" block loading={submitting} disabled={!canSubmit}>
		{submitting ? t('login.code_submitting') : t('login.code_submit')}
	</Button>

	<div class="again">
		<button type="button" onclick={resend} disabled={countdown > 0 || resending}>
			{countdown > 0 ? t('login.code_resend_in', { seconds: countdown }) : t('login.code_resend')}
		</button>
	</div>
</Form>

<style>
	.again {
		text-align: center;
	}

	.again button {
		padding: 0;
		border: none;
		background: none;
		color: var(--color-primary);
		font: inherit;
		font-size: var(--text-sm);
		cursor: pointer;
	}

	.again button:disabled {
		color: var(--color-text-muted);
		cursor: default;
	}

	.again button:not(:disabled):hover {
		text-decoration: underline;
	}
</style>
