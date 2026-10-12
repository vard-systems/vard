<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Checkbox } from '$lib/components/ui';

	type Props = {
		name: string;
		description?: string;
		checked: boolean;
		onChange: (checked: boolean) => void;
		disabled?: boolean;
		reason?: string;
		badges?: Snippet;
	};

	let { name, description, checked, onChange, disabled, reason, badges }: Props = $props();
</script>

<div class="choice" class:checked class:disabled>
	<Checkbox {checked} {onChange} {disabled} title={name} />

	<button
		type="button"
		class="body"
		{disabled}
		onclick={() => onChange(!checked)}
		aria-hidden="true"
		tabindex="-1"
	>
		<span class="name">{name}</span>
		{#if description}
			<span class="description">{description}</span>
		{/if}
		{#if reason}
			<span class="reason">{reason}</span>
		{/if}
	</button>

	{#if badges}
		<span class="badges">{@render badges()}</span>
	{/if}
</div>

<style>
	.choice {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		padding: var(--space-2);
		border-radius: var(--radius-sm);
		transition: background-color var(--speed-fast);
	}

	.choice:hover:not(.disabled) {
		background: var(--row-hover);
	}

	.choice :global([data-scope='checkbox'][data-part='root']) {
		padding-top: 1px;
	}

	.body {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
		padding: 0;
		border: none;
		background: transparent;
		color: inherit;
		font: inherit;
		text-align: left;
		cursor: pointer;
	}

	.disabled .body {
		cursor: not-allowed;
	}

	.name {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 600;
		line-height: 20px;
	}

	.disabled .name {
		color: var(--color-text-hint);
	}

	.description,
	.reason {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.4;
	}

	.reason {
		font-style: italic;
	}

	.badges {
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: var(--space-1);
	}
</style>
