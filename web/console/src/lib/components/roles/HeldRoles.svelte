<script lang="ts">
	import type { Admin, RoleMapping } from '#lib/api/index.js';
	import { Tooltip } from '#lib/components/ui/index.js';
	import { explain, type ScopeRow } from './mappings';
	import { canAssign } from './roles';

	type Props = {
		rows: ScopeRow[];
		held: RoleMapping[];
		/** How many are inherited, for the line shown when only they exist. */
		inherited: number;
		admin: Admin;
		busy: boolean;
		/** The role being removed, drawn faded until it goes. */
		removing: string | null;
		onremove: (role: RoleMapping) => void;
	};

	let { rows, held, inherited, admin, busy, removing, onremove }: Props = $props();
</script>

<div class="panel">
	{#each rows as row (row.key)}
		<div class="row">
			<div class="scope" title={row.clientId}>
				<i
					class={row.key === 'global' ? 'ri-global-line' : 'ri-shield-user-line'}
					style:font-size="15px"
					aria-hidden="true"
				></i>
				<span>{row.name}</span>
			</div>

			<ul class="chips">
				{#each row.roles as role (role.id)}
					{@const removable =
						role.assigned && canAssign(admin, { application_id: role.application?.id ?? null })}
					<li>
						<Tooltip label={explain(role, held)} placement="top">
							{#snippet children(trigger)}
								<span
									{...trigger()}
									class="chip"
									class:inherited={!role.assigned}
									class:removable
									class:leaving={removing === role.id}
								>
									{#if role.composite}
										<i class="ri-node-tree" style:font-size="13px" aria-hidden="true"></i>
									{/if}
									{role.name}
									{#if removable}
										<button
											type="button"
											class="remove"
											aria-label="Remove {role.name}"
											onclick={() => onremove(role)}
											disabled={busy}
										>
											<i class="ri-close-line" style:font-size="14px" aria-hidden="true"></i>
										</button>
									{/if}
								</span>
							{/snippet}
						</Tooltip>
					</li>
				{/each}
			</ul>
		</div>
	{:else}
		<p class="none">
			No roles were given directly. Turn on <em>Show inherited</em> to see the
			{inherited} that come with other roles.
		</p>
	{/each}
</div>

<div class="legend">
	<span><span class="swatch"></span>Assigned</span>
	{#if inherited > 0}
		<span><span class="swatch inherited"></span>Inherited — comes with a composite role</span>
	{/if}
	<span
		><i class="ri-node-tree" style:font-size="13px" aria-hidden="true"></i>Composite — includes
		other roles</span
	>
</div>

<style>
	.panel {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.row {
		display: grid;
		grid-template-columns: 9rem 1fr;
		align-items: start;
		gap: var(--space-3);
		padding: 10px var(--space-3);
	}

	.row + .row {
		border-top: 1px solid var(--color-border);
	}

	.scope {
		display: flex;
		align-items: center;
		gap: 6px;
		min-width: 0;
		height: 28px;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.scope span {
		overflow: hidden;
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
		gap: 5px;
		height: 28px;
		padding: 0 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 500;
		white-space: nowrap;
		cursor: default;
		transition: opacity var(--speed-fast);
	}

	.chip.removable {
		padding-right: 3px;
	}

	.chip.inherited {
		border-style: dashed;
		background: transparent;
		color: var(--color-text-hint);
	}

	.chip.leaving {
		opacity: 0.45;
	}

	.remove {
		display: grid;
		place-items: center;
		width: 22px;
		height: 22px;
		padding: 0;
		border: none;
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--color-text-hint);
		cursor: pointer;
		transition:
			background-color var(--speed-fast),
			color var(--speed-fast);
	}

	.remove:hover:not(:disabled) {
		background: var(--surface-danger);
		color: var(--color-danger);
	}

	.remove:focus-visible {
		outline: 2px solid var(--color-info);
	}

	.none {
		margin: 0;
		padding: var(--space-3);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1) var(--space-4);
		margin-top: var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.legend > span {
		display: inline-flex;
		align-items: center;
		gap: 6px;
	}

	.swatch {
		width: 16px;
		height: 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
	}

	.swatch.inherited {
		border-style: dashed;
		background: transparent;
	}

	@media (max-width: 34rem) {
		.row {
			grid-template-columns: 1fr;
			gap: var(--space-2);
		}

		.scope {
			height: auto;
		}
	}
</style>
