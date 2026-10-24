<script lang="ts">
	import type { API } from '#lib/api/index.js';
	import { Button, FormSection, SearchInput } from '#lib/components/ui/index.js';
	import AddPanel from './AddPanel.svelte';
	import Choice from './Choice.svelte';

	type Props = {
		/** The ids of the API scopes the role grants. */
		granted: string[];
		apis: API[];
		editable: boolean;
	};

	let { granted = $bindable(), apis, editable }: Props = $props();

	let adding = $state(false);
	let picked = $state<string[]>([]);
	let search = $state('');

	const grantedByApi = $derived(
		apis
			.map((api) => ({ api, scopes: api.scopes.filter((scope) => granted.includes(scope.id)) }))
			.filter((group) => group.scopes.length > 0)
	);

	const candidates = $derived.by(() => {
		const wanted = search.trim().toLowerCase();

		return apis
			.map((api) => ({
				api,
				scopes: api.scopes.filter(
					(scope) =>
						!granted.includes(scope.id) &&
						(wanted === '' ||
							[scope.name, scope.description, api.name, api.identifier].some((text) =>
								text.toLowerCase().includes(wanted)
							))
				)
			}))
			.filter((group) => group.scopes.length > 0);
	});

	function pick(id: string, on: boolean) {
		picked = on ? [...picked, id] : picked.filter((other) => other !== id);
	}

	function close() {
		adding = false;
		picked = [];
	}

	function add() {
		granted = [...granted, ...picked];
		close();
	}
</script>

<FormSection
	title="API permissions"
	description="Scopes this role grants users. A token carries one only for its own API, and only when the application is allowed it too."
>
	{#snippet action()}
		{#if editable && !adding}
			<Button size="sm" variant="subtle" onclick={() => (adding = true)}>
				<i class="ri-add-line" aria-hidden="true"></i>
				Add scopes
			</Button>
		{/if}
	{/snippet}

	{#if adding}
		<AddPanel count={picked.length} oncancel={close} onadd={add}>
			<SearchInput
				label="Search API scopes"
				placeholder="Search by scope or API…"
				size="sm"
				bind:value={search}
				onkeydown={(event) => event.key === 'Enter' && event.preventDefault()}
			/>

			<div class="list">
				{#each candidates as group (group.api.id)}
					<div class="group">
						<span class="api">
							{group.api.name}
							<code>{group.api.identifier}</code>
						</span>
						{#each group.scopes as scope (scope.id)}
							<Choice
								name={scope.name}
								description={scope.description}
								checked={picked.includes(scope.id)}
								onChange={(on) => pick(scope.id, on)}
							/>
						{/each}
					</div>
				{:else}
					<p class="none">
						{apis.length === 0
							? 'No APIs are registered yet.'
							: 'No more scopes to add, or none match this.'}
					</p>
				{/each}
			</div>
		</AddPanel>
	{/if}

	{#if grantedByApi.length > 0}
		<div class="grants">
			{#each grantedByApi as group (group.api.id)}
				<div class="row">
					<span class="row-api" title={group.api.identifier}>{group.api.name}</span>
					<ul class="chips">
						{#each group.scopes as scope (scope.id)}
							<li class="chip" title={scope.description}>
								{scope.name}
								{#if editable}
									<button
										type="button"
										aria-label="Stop granting {scope.name}"
										onclick={() => (granted = granted.filter((id) => id !== scope.id))}
									>
										<i class="ri-close-line" style:font-size="13px" aria-hidden="true"></i>
									</button>
								{/if}
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</div>
	{:else if !adding}
		<p class="none">This role grants no API scopes.</p>
	{/if}
</FormSection>

<style>
	.none {
		margin: 0;
		padding: var(--space-3);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-surface);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		text-align: center;
	}

	.list {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		max-height: 20rem;
		overflow-y: auto;
	}

	.group {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.api {
		display: flex;
		flex-wrap: wrap;
		align-items: baseline;
		gap: var(--space-2);
		padding: 0 var(--space-2);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.api code {
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-xs);
		font-weight: 400;
	}

	.grants {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.row {
		display: grid;
		grid-template-columns: 8rem 1fr;
		align-items: start;
		gap: var(--space-3);
		padding: 10px var(--space-3);
	}

	.row + .row {
		border-top: 1px solid var(--color-border);
	}

	.row-api {
		overflow: hidden;
		line-height: 26px;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-weight: 600;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.chip {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		height: 26px;
		padding: 0 3px 0 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.chip button {
		display: grid;
		place-items: center;
		width: 20px;
		height: 20px;
		padding: 0;
		border: none;
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--color-text-hint);
		cursor: pointer;
	}

	.chip button:hover {
		background: var(--surface-danger);
		color: var(--color-danger);
	}

	@media (max-width: 30rem) {
		.row {
			grid-template-columns: 1fr;
			gap: var(--space-2);
		}
	}
</style>
