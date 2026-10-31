<script lang="ts">
	import { useTranslator } from '#lib/i18n/index.js';

	type Props = {
		label: string;
		value: string;
		autocomplete?: 'current-password' | 'new-password';
		disabled?: boolean;
		hint?: string;
		error?: string;
		aside?: { label: string; href: string };
	};

	let {
		label,
		value = $bindable(''),
		autocomplete = 'current-password',
		disabled = false,
		hint,
		error,
		aside
	}: Props = $props();

	const t = useTranslator();

	let visible = $state(false);

	const uid = $props.id();
	const inputId = `password-${uid}`;
	const noteId = `${inputId}-note`;
</script>

<div class="field">
	<div class="top">
		<label for={inputId}>{label}</label>
		{#if aside}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- built with resolve() by the page -->
			<a href={aside.href}>{aside.label}</a>
		{/if}
	</div>

	<div class="control">
		<input
			id={inputId}
			type={visible ? 'text' : 'password'}
			bind:value
			{autocomplete}
			{disabled}
			autocapitalize="none"
			spellcheck="false"
			aria-invalid={error ? 'true' : undefined}
			aria-describedby={error || hint ? noteId : undefined}
		/>
		<button
			type="button"
			class="reveal"
			onclick={() => (visible = !visible)}
			aria-label={visible ? t('field.hide_password') : t('field.show_password')}
			aria-pressed={visible}
			{disabled}
		>
			{#if visible}<i class="ri-eye-off-line" aria-hidden="true"></i>{:else}<i
					class="ri-eye-line"
					aria-hidden="true"
				></i>{/if}
		</button>
	</div>

	{#if error}
		<p id={noteId} class="note error">{error}</p>
	{:else if hint}
		<p id={noteId} class="note">{hint}</p>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.top {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: var(--space-2);
	}

	label {
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.top a {
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.control {
		position: relative;
	}

	input {
		width: 100%;
		height: var(--control-height);
		padding: 0 48px 0 var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-input);
		font-size: var(--text-lg);
		transition:
			border-color var(--speed),
			background var(--speed),
			box-shadow var(--speed);
	}

	input:hover:not(:disabled) {
		background: var(--color-input-hover);
	}

	input:focus {
		outline: none;
		border-color: var(--color-focus);
		background: var(--color-surface);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-focus), transparent 80%);
	}

	input[aria-invalid='true'] {
		border-color: var(--color-danger);
	}

	.reveal {
		position: absolute;
		top: 50%;
		right: 6px;
		display: grid;
		place-items: center;
		width: 36px;
		height: 36px;
		border: 0;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text-hint);
		cursor: pointer;
		transform: translateY(-50%);
	}

	.reveal:hover {
		color: var(--color-text);
	}

	.note {
		font-size: var(--text-sm);
		color: var(--color-text-hint);
	}

	.note.error {
		color: var(--color-danger);
	}
</style>
