<script lang="ts">
	import { RiAddLine, RiCloseLine } from '#lib/icons.js';
	import { Button, Icon, Input } from '#lib/components/ui/index.js';

	type Props = {
		domains: string[];
		readOnly?: boolean;
	};

	let { domains = $bindable([]), readOnly = false }: Props = $props();

	let typed = $state('');

	function normalize(domain: string): string {
		return domain.trim().toLowerCase().replace(/^@/, '').replace(/\.$/, '');
	}

	function add() {
		for (const piece of typed.split(/[\s,;]+/)) {
			const domain = normalize(piece);
			if (domain && !domains.includes(domain)) domains = [...domains, domain];
		}
		typed = '';
	}

	function remove(domain: string) {
		domains = domains.filter((one) => one !== domain);
	}

	/** Enter adds rather than submitting the dialog's form. */
	function keydown(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			add();
		}
	}
</script>

<div class="field">
	{#if domains.length > 0}
		<ul class="chips">
			{#each domains as domain (domain)}
				<li>
					<span>{domain}</span>
					{#if !readOnly}
						<button type="button" aria-label="Remove {domain}" onclick={() => remove(domain)}>
							<Icon icon={RiCloseLine} size="0.9rem" />
						</button>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if !readOnly}
		<div class="add">
			<Input
				label="Add a domain"
				bind:value={typed}
				placeholder="acme.com"
				hint="People with an address at these domains sign in through this connection, and it signs in nobody else. Leave it empty to trust the provider with any address but those at another connection's domains; people then reach it through its button only."
				onkeydown={keydown}
				autocomplete="off"
				spellcheck={false}
			/>
			<Button variant="subtle" onclick={add} disabled={typed.trim() === ''}>
				<Icon icon={RiAddLine} />
				Add
			</Button>
		</div>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		display: inline-flex;
		align-items: center;
		gap: 2px;
		padding: 3px 4px 3px 8px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	li button {
		display: grid;
		place-items: center;
		width: 20px;
		height: 20px;
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text-hint);
		cursor: pointer;
	}

	li button:hover {
		background: var(--color-secondary);
		color: var(--color-text);
	}

	.add {
		display: grid;
		grid-template-columns: 1fr auto;
		align-items: start;
		gap: var(--space-2);
	}

	.add :global(button) {
		margin-top: 6px;
	}
</style>
