<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { IconName } from './icon';

	type Props = {
		icon: IconName;
		title: Snippet;
		detail?: string;
		actions?: Snippet;
		/** Under the row, the width of it: a form the row opened. */
		below?: Snippet;
	};

	let { icon, title, detail, actions, below }: Props = $props();
</script>

<li>
	<div class="row">
		<span class="tile"><i class={icon} aria-hidden="true"></i></span>
		<div class="details">
			<strong>{@render title()}</strong>
			{#if detail}<span>{detail}</span>{/if}
		</div>
		{#if actions}<div class="actions">{@render actions()}</div>{/if}
	</div>
	{#if below}<div class="below">{@render below()}</div>{/if}
</li>

<style>
	.row {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.tile {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		width: 40px;
		height: 40px;
		border-radius: var(--radius-md);
		background: var(--color-input);
		color: var(--color-text-hint);
	}

	.details {
		display: flex;
		flex: 1;
		flex-direction: column;
		min-width: 0;
	}

	strong {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
	}

	.details > span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}

	.actions {
		display: flex;
		flex-shrink: 0;
		gap: var(--space-2);
	}

	.below {
		margin-top: var(--space-3);
		padding-left: calc(40px + var(--space-3));
	}

	@media (max-width: 30rem) {
		.below {
			padding-left: 0;
		}
	}
</style>
