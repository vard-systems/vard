<script lang="ts">
	import type { IconName } from './icon';
	import type { Snippet } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';

	type Tab = {
		value: string;
		label: string;
		icon?: IconName;
		count?: number;
		disabled?: boolean;
	};

	type Props = {
		tabs: Tab[];
		value: string;
		label: string;
		panel: Snippet<[string]>;
	};

	let { tabs, value = $bindable(), label, panel }: Props = $props();

	const id = $props.id();

	const opened = new SvelteSet<string>();

	$effect(() => {
		opened.add(value);
	});

	let triggers: HTMLButtonElement[] = $state([]);

	function keydown(event: KeyboardEvent, index: number) {
		const enabled = tabs.map((tab, i) => (tab.disabled ? -1 : i)).filter((i) => i >= 0);
		const at = enabled.indexOf(index);

		let next: number | undefined;
		if (event.key === 'ArrowRight') next = enabled[(at + 1) % enabled.length];
		if (event.key === 'ArrowLeft') next = enabled[(at - 1 + enabled.length) % enabled.length];
		if (event.key === 'Home') next = enabled[0];
		if (event.key === 'End') next = enabled[enabled.length - 1];

		if (next === undefined) return;

		event.preventDefault();
		value = tabs[next].value;
		triggers[next]?.focus();
	}
</script>

<div class="tabs">
	<div class="list" role="tablist" aria-label={label}>
		{#each tabs as tab, index (tab.value)}
			<button
				bind:this={triggers[index]}
				type="button"
				role="tab"
				id="{id}-tab-{tab.value}"
				aria-controls="{id}-panel-{tab.value}"
				aria-selected={tab.value === value}
				tabindex={tab.value === value ? 0 : -1}
				disabled={tab.disabled}
				class:selected={tab.value === value}
				onclick={() => (value = tab.value)}
				onkeydown={(event) => keydown(event, index)}
			>
				{#if tab.icon}
					<i class={tab.icon} aria-hidden="true"></i>
				{/if}
				{tab.label}
				{#if tab.count !== undefined}
					<span class="count">{tab.count}</span>
				{/if}
			</button>
		{/each}
	</div>

	{#each tabs as tab (tab.value)}
		{#if opened.has(tab.value) || tab.value === value}
			<div
				class="panel"
				role="tabpanel"
				id="{id}-panel-{tab.value}"
				aria-labelledby="{id}-tab-{tab.value}"
				hidden={tab.value !== value}
			>
				{@render panel(tab.value)}
			</div>
		{/if}
	{/each}
</div>

<style>
	.list {
		display: flex;
		gap: var(--space-1);
		overflow-x: auto;
		box-shadow: inset 0 -1px 0 var(--color-border);
		scrollbar-width: none;
	}

	.list::-webkit-scrollbar {
		display: none;
	}

	button {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		flex-shrink: 0;
		height: 42px;
		padding: 0 var(--space-3);
		border: none;
		background: transparent;
		color: var(--color-text-hint);
		font: inherit;
		font-size: var(--text-base);
		font-weight: 600;
		white-space: nowrap;
		cursor: pointer;
		transition:
			color var(--speed-fast),
			box-shadow var(--speed-fast);
	}

	button:hover:not(:disabled) {
		color: var(--color-text);
	}

	button.selected {
		box-shadow: inset 0 -2px 0 var(--color-text);
		color: var(--color-text);
	}

	button:disabled {
		cursor: not-allowed;
		opacity: 0.5;
	}

	button:focus-visible {
		border-radius: var(--radius-sm) var(--radius-sm) 0 0;
		outline: 2px solid var(--color-info);
		outline-offset: -2px;
	}

	.count {
		display: inline-grid;
		place-items: center;
		min-width: 20px;
		height: 20px;
		padding: 0 6px;
		border-radius: var(--radius-pill);
		background: var(--color-secondary);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.panel {
		padding-top: var(--space-5);
	}
</style>
