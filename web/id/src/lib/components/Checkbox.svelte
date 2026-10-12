<script lang="ts">
	import type { Snippet } from 'svelte';
	import Icon from './Icon.svelte';

	type Props = {
		checked: boolean;
		disabled?: boolean;
		id?: string;
		name?: string;
		children: Snippet;
	};

	let { checked = $bindable(false), disabled = false, id, name, children }: Props = $props();

	const uid = $props.id();
</script>

<label class="checkbox" class:disabled>
	<input id={id ?? `checkbox-${uid}`} {name} type="checkbox" bind:checked {disabled} />
	<span class="box" aria-hidden="true"><Icon name="check" size="0.875rem" /></span>
	<span class="text">{@render children()}</span>
</label>

<style>
	.checkbox {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		font-size: var(--text-base);
		line-height: 1.5;
		cursor: pointer;
	}

	.disabled {
		cursor: not-allowed;
		opacity: 0.6;
	}

	input {
		position: absolute;
		opacity: 0;
		width: 1px;
		height: 1px;
	}

	.box {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		width: 20px;
		height: 20px;
		margin-top: 1px;
		border: 1.5px solid var(--color-border-strong);
		border-radius: 5px;
		background: var(--color-surface);
		color: transparent;
		transition:
			background var(--speed),
			border-color var(--speed);
	}

	input:checked + .box {
		border-color: var(--color-primary);
		background: var(--color-primary);
		color: var(--color-primary-text);
	}

	input:focus-visible + .box {
		outline: 2px solid var(--color-focus);
		outline-offset: 2px;
	}

	.text :global(a) {
		font-weight: 600;
	}
</style>
