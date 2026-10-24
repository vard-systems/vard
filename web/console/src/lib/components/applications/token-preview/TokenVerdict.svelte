<script lang="ts">
	import type { TokenPreview } from '#lib/api/index.js';
	import { CodeBlock } from '#lib/components/ui/index.js';

	type Props = { preview: TokenPreview };

	let { preview }: Props = $props();

	const granted = $derived(preview.decisions.filter((it) => it.granted).length);

	const blocks = $derived(
		[
			{ title: 'Access token header', value: preview.access_token_header },
			{ title: 'Access token claims', value: preview.access_token },
			{ title: 'ID token claims', value: preview.id_token }
		].filter((block) => block.value)
	);
</script>

<div class="result">
	<div class="verdict" class:issued={preview.issued}>
		<i
			class={preview.issued ? 'ri-check-line' : 'ri-close-line'}
			style:font-size="18px"
			aria-hidden="true"
		></i>
		<div>
			<strong>{preview.issued ? 'A token would be issued' : 'No token would be issued'}</strong>
			{#if preview.reason}
				<p>{preview.reason}</p>
			{:else if preview.issued}
				<p>
					{granted} of {preview.decisions.length}
					requested {preview.decisions.length === 1 ? 'scope' : 'scopes'} granted.
				</p>
			{/if}
		</div>
	</div>

	{#if preview.decisions.length > 0}
		<ul class="decisions">
			{#each preview.decisions as decision (decision.scope)}
				<li class:granted={decision.granted}>
					<i
						class={decision.granted ? 'ri-check-line' : 'ri-close-line'}
						style:font-size="15px"
						aria-hidden="true"
					></i>
					<code>{decision.scope}</code>
					<span class="kind">
						{decision.is_default ? 'Default' : decision.kind === 'api' ? 'API' : 'OpenID'}
					</span>
					<span class="reason">{decision.reason}</span>
				</li>
			{/each}
		</ul>
	{/if}

	{#each blocks as block (block.title)}
		<CodeBlock title={block.title} code={JSON.stringify(block.value, null, 2)} />
	{/each}
</div>

<style>
	.result {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		margin-top: var(--space-4);
	}

	.verdict {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		padding: var(--space-3);
		border-radius: var(--radius-md);
		background: var(--surface-danger);
		color: var(--color-danger);
	}

	.verdict.issued {
		background: var(--surface-success);
		color: var(--color-success);
	}

	.verdict strong {
		color: var(--color-text);
	}

	.verdict p {
		margin: 2px 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.decisions {
		margin: 0;
		padding: 0;
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		list-style: none;
	}

	.decisions li {
		display: grid;
		grid-template-columns: auto minmax(8rem, auto) auto 1fr;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		color: var(--color-danger);
	}

	.decisions li.granted {
		color: var(--color-success);
	}

	.decisions li + li {
		border-top: 1px solid var(--color-border);
	}

	.decisions code {
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.kind {
		padding: 1px 6px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.reason {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	@media (max-width: 34rem) {
		.decisions li {
			grid-template-columns: auto 1fr auto;
		}

		.reason {
			grid-column: 2 / -1;
		}
	}
</style>
