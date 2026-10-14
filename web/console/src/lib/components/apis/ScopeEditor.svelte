<script lang="ts">
	import { RiAddLine, RiDeleteBinLine } from 'svelte-remixicon';
	import { Button, Checkbox, Icon, IconButton } from '#lib/components/ui/index.js';
	import { scopeProblem, scopeRow, type ScopeRow } from './scopes';

	type Props = {
		rows: ScopeRow[];
		editable: boolean;
	};

	let { rows = $bindable(), editable }: Props = $props();

	function add() {
		rows = [...rows, scopeRow()];
	}
</script>

<div class="editor">
	{#if rows.length > 0}
		<div class="table" role="table" aria-label="Scopes">
			<div class="head" role="row">
				<span role="columnheader">Scope</span>
				<span role="columnheader">Description</span>
				<span role="columnheader" class="center" title="Added to every token for the API">
					Default
				</span>
				<span role="columnheader"></span>
			</div>

			{#each rows as row (row.key)}
				{@const issue = scopeProblem(rows, row)}
				<div class="row" class:invalid={issue !== undefined} role="row">
					<input
						id={`scope-name-${row.key}`}
						name={`scope-name-${row.key}`}
						class="name"
						bind:value={row.name}
						placeholder="orders:read"
						aria-label="Scope name"
						autocapitalize="none"
						spellcheck="false"
						readonly={!editable}
						onkeydown={(event) => event.key === 'Enter' && event.preventDefault()}
					/>
					<input
						id={`scope-description-${row.key}`}
						name={`scope-description-${row.key}`}
						bind:value={row.description}
						placeholder="What it allows"
						aria-label="Scope description"
						readonly={!editable}
						onkeydown={(event) => event.key === 'Enter' && event.preventDefault()}
					/>
					<span class="center">
						<Checkbox
							checked={row.is_default}
							onChange={(on) => (row.is_default = on)}
							disabled={!editable}
							title="Default scope: {row.name || 'new scope'}"
						/>
					</span>
					<span>
						{#if editable}
							<IconButton
								icon={RiDeleteBinLine}
								label="Remove {row.name || 'this scope'}"
								size="sm"
								colorPalette="danger"
								placement="left"
								onclick={() => (rows = rows.filter((it) => it.key !== row.key))}
							/>
						{/if}
					</span>
					{#if issue}
						<span class="issue">{issue}</span>
					{/if}
				</div>
			{/each}
		</div>
	{:else}
		<p class="none">No scopes yet. Add the first, such as <code>orders:read</code>.</p>
	{/if}

	{#if editable}
		<div>
			<Button size="sm" variant="subtle" onclick={add}>
				<Icon icon={RiAddLine} />
				Add scope
			</Button>
		</div>
	{/if}
</div>

<style>
	.editor {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.table {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.head,
	.row {
		display: grid;
		grid-template-columns: minmax(9rem, 2fr) minmax(8rem, 3fr) 4.5rem 36px;
		align-items: center;
		gap: var(--space-2);
		padding: 6px 6px 6px var(--space-2);
	}

	.head {
		padding-block: 8px;
		border-bottom: 1px solid var(--color-border);
		background: var(--color-secondary-alt);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
		font-weight: 700;
	}

	.row + .row {
		border-top: 1px solid var(--color-border);
	}

	.row.invalid {
		background: var(--surface-danger);
	}

	.center {
		display: flex;
		justify-content: center;
		text-align: center;
	}

	input {
		min-width: 0;
		height: 34px;
		padding: 0 10px;
		border: 1px solid var(--color-input-border);
		border-radius: var(--radius-field);
		background: var(--color-input);
		color: var(--color-text);
		font: inherit;
		font-size: var(--text-sm);
	}

	input:focus {
		border-color: var(--color-brand);
		outline: none;
		box-shadow: inset 0 0 0 1px var(--color-brand);
	}

	input[readonly] {
		background: transparent;
	}

	.name {
		font-family: var(--font-mono);
	}

	.issue {
		grid-column: 1 / -1;
		padding: 0 2px 2px;
		color: var(--color-danger);
		font-size: var(--text-xs);
	}

	.none {
		margin: 0;
		padding: var(--space-3);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-surface);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		text-align: center;
	}

	code {
		font-family: var(--font-mono);
	}

	@media (max-width: 34rem) {
		.head {
			display: none;
		}

		.row {
			grid-template-columns: 1fr auto auto;
		}

		.row input:nth-of-type(2) {
			grid-column: 1 / -1;
			grid-row: 2;
		}
	}
</style>
