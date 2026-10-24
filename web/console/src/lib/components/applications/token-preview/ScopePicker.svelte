<script lang="ts">
	type Props = {
		/** The scopes the application and the chosen API offer, as chips. */
		offered: string[];
		chosen: string[];
		/** Anything else to ask for, typed: a scope may be asked for that is not offered. */
		extra: string;
		/** The scope parameter the request would send. */
		param: string;
		/** Enter in the text box, which would otherwise submit the dialog's form. */
		onenter: () => void;
	};

	let { offered, chosen = $bindable([]), extra = $bindable(''), param, onenter }: Props = $props();

	function toggle(scope: string) {
		chosen = chosen.includes(scope) ? chosen.filter((it) => it !== scope) : [...chosen, scope];
	}
</script>

<div class="field">
	<span class="label">Scopes</span>
	{#if offered.length > 0}
		<div class="chips">
			{#each offered as scope (scope)}
				<button
					type="button"
					class:on={chosen.includes(scope)}
					aria-pressed={chosen.includes(scope)}
					onclick={() => toggle(scope)}
				>
					{#if chosen.includes(scope)}
						<i class="ri-check-line" style:font-size="13px" aria-hidden="true"></i>
					{/if}
					{scope}
				</button>
			{/each}
		</div>
	{/if}
	<input
		id="token-preview-extra-scopes"
		name="token-preview-extra-scopes"
		class="extra"
		bind:value={extra}
		placeholder="Other scopes, separated by spaces"
		onkeydown={(event) => {
			if (event.key === 'Enter') {
				event.preventDefault();
				onenter();
			}
		}}
		aria-label="Other scopes"
		autocapitalize="none"
		spellcheck="false"
	/>
	<code class="param">scope={param || '(none)'}</code>
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.label {
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.chips button {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		height: 28px;
		padding: 0 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		cursor: pointer;
	}

	.chips button.on {
		border-color: var(--color-text);
		background: var(--color-text);
		color: var(--color-surface);
	}

	.extra {
		height: 36px;
		padding: 0 12px;
		border: 1px solid var(--color-input-border);
		border-radius: var(--radius-sm);
		background: var(--color-input);
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.extra:focus {
		border-color: var(--color-brand);
		outline: none;
		box-shadow: inset 0 0 0 1px var(--color-brand);
	}

	.param {
		overflow-wrap: anywhere;
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-xs);
	}
</style>
