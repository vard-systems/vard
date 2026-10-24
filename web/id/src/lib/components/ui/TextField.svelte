<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	type Props = Omit<HTMLInputAttributes, 'value'> & {
		label: string;
		value: string;
		hint?: string;
		error?: string;
	};

	let { label, value = $bindable(''), hint, error, id, ...input }: Props = $props();

	const uid = $props.id();
	const inputId = $derived(id ?? `field-${uid}`);
	const noteId = $derived(`${inputId}-note`);
</script>

<div class="field">
	<label for={inputId}>{label}</label>
	<input
		id={inputId}
		bind:value
		aria-invalid={error ? 'true' : undefined}
		aria-describedby={error || hint ? noteId : undefined}
		{...input}
	/>
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
		min-width: 0;
	}

	label {
		font-size: var(--text-sm);
		font-weight: 600;
	}

	input {
		width: 100%;
		height: var(--control-height);
		padding: 0 var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-input);
		font-size: var(--text-lg);
		transition:
			border-color var(--speed),
			background var(--speed),
			box-shadow var(--speed);
	}

	input:hover:not(:disabled):not(:read-only) {
		background: var(--color-input-hover);
	}

	input:focus {
		outline: none;
		border-color: var(--color-focus);
		background: var(--color-surface);
		box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-focus), transparent 80%);
	}

	input:read-only {
		color: var(--color-text-hint);
	}

	input:disabled {
		opacity: 0.6;
	}

	input[aria-invalid='true'] {
		border-color: var(--color-danger);
	}

	.note {
		font-size: var(--text-sm);
		color: var(--color-text-hint);
	}

	.note.error {
		color: var(--color-danger);
	}
</style>
