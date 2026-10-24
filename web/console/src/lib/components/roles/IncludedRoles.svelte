<script lang="ts">
	import type { Admin, Application, Role } from '#lib/api/index.js';
	import { Badge, Button, FormSection, IconButton } from '#lib/components/ui/index.js';
	import AddPanel from './AddPanel.svelte';
	import RolePicker from './RolePicker.svelte';
	import { canEditRoles, mayInherit, reachable, scopeName, wouldCycle } from './roles';

	type Props = {
		/** The ids of the roles this one includes. */
		inherits: string[];
		/** The role being edited; null while a new one is being made. */
		role: Role | null;
		/** Where the role applies: null for global, undefined until it is chosen. */
		scope: string | null | undefined;
		roles: Role[];
		applications: Application[];
		admin: Admin;
		editable: boolean;
	};

	let {
		inherits = $bindable(),
		role,
		scope,
		roles,
		applications,
		admin,
		editable
	}: Props = $props();

	let adding = $state(false);
	let picked = $state<string[]>([]);

	const direct = $derived(roles.filter((other) => inherits.includes(other.id)));

	const effective = $derived(
		roles.filter((other) => reachable(inherits, roles).has(other.id) && other.id !== role?.id)
	);

	const candidates = $derived(
		scope === undefined
			? []
			: roles.filter(
					(other) =>
						other.id !== role?.id &&
						!inherits.includes(other.id) &&
						mayInherit(scope, other) &&
						canEditRoles(admin, other.application_id)
				)
	);

	function label(other: Role): string {
		return other.application_id === scope
			? other.name
			: `${scopeName(other, applications)} / ${other.name}`;
	}

	function close() {
		adding = false;
		picked = [];
	}

	function add() {
		inherits = [...inherits, ...picked];
		close();
	}
</script>

<FormSection
	title="Included roles"
	description={scope === null
		? 'Anyone with this role also gets the roles it includes. A global role can include any role.'
		: 'Anyone with this role also gets the roles it includes: global roles, or roles of the same application.'}
>
	{#snippet action()}
		{#if editable && !adding}
			<Button
				size="sm"
				variant="subtle"
				onclick={() => (adding = true)}
				disabled={scope === undefined}
			>
				<i class="ri-add-line" aria-hidden="true"></i>
				Add roles
			</Button>
		{/if}
	{/snippet}

	{#if adding}
		<AddPanel count={picked.length} oncancel={close} onadd={add}>
			<RolePicker
				roles={candidates}
				{applications}
				bind:picked
				reason={(other) =>
					wouldCycle(role?.id, other.id, roles)
						? `Already includes ${role?.name}, so this would go round in a circle.`
						: undefined}
				empty="There are no other roles this role can include."
			/>
		</AddPanel>
	{/if}

	{#if direct.length > 0}
		<ul class="included">
			{#each direct as other (other.id)}
				<li>
					<i
						class={other.application_id === null ? 'ri-global-line' : 'ri-shield-user-line'}
						style:font-size="15px"
						aria-hidden="true"
					></i>
					<span class="name">{label(other)}</span>
					<span class="description">{other.description}</span>
					{#if editable}
						<IconButton
							icon="ri-close-line"
							label="Stop including {other.name}"
							size="sm"
							placement="left"
							onclick={() => (inherits = inherits.filter((id) => id !== other.id))}
						/>
					{/if}
				</li>
			{/each}
		</ul>
	{:else if !adding}
		<p class="none">This role does not include other roles.</p>
	{/if}

	{#if effective.length > direct.length}
		<div class="effective">
			<span>Counting what those include, users also get</span>
			<div class="badges">
				{#each effective as other (other.id)}
					<Badge>{label(other)}</Badge>
				{/each}
			</div>
		</div>
	{/if}
</FormSection>

<style>
	.included {
		margin: 0;
		padding: 0;
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		list-style: none;
	}

	.included li {
		display: grid;
		grid-template-columns: auto auto 1fr auto;
		align-items: center;
		gap: var(--space-2);
		min-height: 44px;
		padding: 0 var(--space-2) 0 var(--space-3);
		color: var(--color-text-hint);
	}

	.included li + li {
		border-top: 1px solid var(--color-border);
	}

	.name {
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 600;
		white-space: nowrap;
	}

	.description {
		overflow: hidden;
		font-size: var(--text-sm);
		text-overflow: ellipsis;
		white-space: nowrap;
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

	.effective {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.badges {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}

	@media (max-width: 30rem) {
		.included li {
			grid-template-columns: auto 1fr auto;
		}

		.description {
			display: none;
		}
	}
</style>
