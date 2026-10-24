<script lang="ts">
	import type { Organization, OrganizationSettings } from '#lib/api/index.js';
	import { Code, Tag, Thumb } from '#lib/components/ui/index.js';
	import { formatDate, initials } from '#lib/utils/format.js';
	import { publicDetails } from './settings';

	type Props = {
		/** What the form holds now, so the card follows the typing. */
		input: OrganizationSettings;
		saved: Organization;
	};

	let { input, saved }: Props = $props();

	const logo = $derived(/^https?:\/\/\S+$/.test(input.logo_url) ? input.logo_url : '');
	const details = $derived(publicDetails(input));
	const filled = $derived(details.filter((detail) => detail.filled).length);
	const missing = $derived(details.filter((detail) => !detail.filled));
</script>

<section class="summary" aria-label="Organization">
	<div class="identity">
		<Thumb src={logo} text={initials(input.name)} size="md" shape="circle" tone="accent" />

		<div class="names">
			<strong title={input.name}>{input.name || 'Unnamed organization'}</strong>
			<span class="line">
				<Code tone="quiet" truncate>{input.slug || saved.slug}</Code>
				<span aria-hidden="true">·</span>
				<span title="When this organization's record was created">
					Since {formatDate(saved.created_at)}
				</span>
			</span>
		</div>
	</div>

	<div class="progress">
		<div class="count">
			<span>Public details</span>
			<strong>{filled} of {details.length}</strong>
		</div>
		<div
			class="meter"
			role="meter"
			aria-label="Public details filled in"
			aria-valuemin={0}
			aria-valuemax={details.length}
			aria-valuenow={filled}
		>
			<span style:width="{(filled / details.length) * 100}%"></span>
		</div>
		{#if missing.length > 0}
			<div class="missing">
				{#each missing as detail (detail.label)}<Tag small>{detail.label}</Tag>{/each}
			</div>
			<small>The sign-in pages leave these out until they are filled in.</small>
		{:else}
			<small>Everything the sign-in pages can show is filled in.</small>
		{/if}
	</div>
</section>

<style>
	.summary {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		padding: var(--space-4);
		border: 1px solid var(--color-secondary-alt);
		border-radius: var(--radius-surface);
		background: var(--color-surface);
		box-shadow: var(--shadow-panel);
	}

	.identity {
		display: flex;
		flex: 1 1 18rem;
		align-items: center;
		gap: var(--space-3);
		min-width: 0;
	}

	.names {
		display: flex;
		flex-direction: column;
		gap: 4px;
		min-width: 0;
	}

	.names strong {
		overflow: hidden;
		font-size: var(--text-xl);
		font-weight: 600;
		line-height: 1.25;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.line {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		min-width: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		white-space: nowrap;
	}

	.progress {
		display: flex;
		flex: 0 1 20rem;
		flex-direction: column;
		gap: var(--space-1);
		min-width: 0;
	}

	.count {
		display: flex;
		justify-content: space-between;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.count strong {
		color: var(--color-text);
		font-weight: 600;
	}

	.meter {
		height: 6px;
		overflow: hidden;
		border-radius: var(--radius-pill);
		background: var(--color-secondary);
	}

	.meter span {
		display: block;
		height: 100%;
		border-radius: inherit;
		background: var(--color-brand);
		transition: width var(--speed);
	}

	.missing {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin-top: 2px;
	}

	.progress small {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	@media (prefers-reduced-motion: reduce) {
		.meter span {
			transition: none;
		}
	}
</style>
