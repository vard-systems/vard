<script lang="ts">
	import { RiGlobalLine, RiShieldUserLine } from '#lib/icons.js';
	import type { Application, Role } from '#lib/api/index.js';
	import { Badge, Icon, SearchInput } from '#lib/components/ui/index.js';
	import Choice from './Choice.svelte';
	import { byScope } from './roles';

	type Props = {
		roles: Role[];
		applications: Application[];
		picked: string[];
		reason?: (role: Role) => string | undefined;
		empty?: string;
	};

	let {
		roles,
		applications,
		picked = $bindable([]),
		reason = () => undefined,
		empty = 'There are no roles to choose from.'
	}: Props = $props();

	let search = $state('');

	const groups = $derived(
		byScope(
			roles.filter((role) => {
				const wanted = search.trim().toLowerCase();
				if (wanted === '') return true;

				const app = applications.find((it) => it.id === role.application_id)?.name ?? 'global';

				return [role.name, role.description, app].some((text) =>
					text.toLowerCase().includes(wanted)
				);
			}),
			applications
		)
	);

	function toggle(id: string, on: boolean) {
		picked = on ? [...picked, id] : picked.filter((it) => it !== id);
	}

	/** Enter in the search box would submit the form around the panel. */
	function stayPut(event: KeyboardEvent) {
		if (event.key === 'Enter') event.preventDefault();
	}
</script>

<div class="picker">
	<SearchInput
		label="Search roles"
		placeholder="Search by role or application…"
		bind:value={search}
		onkeydown={stayPut}
	/>

	<div class="list">
		{#each groups as group (group.scope)}
			<div class="group">
				<div class="heading">
					<Icon icon={group.scope === null ? RiGlobalLine : RiShieldUserLine} size="0.875rem" />
					{group.scope === null ? 'Global roles' : group.name}
				</div>

				{#each group.roles as role (role.id)}
					{@const why = reason(role)}
					<Choice
						name={role.name}
						description={why ?? role.description}
						checked={picked.includes(role.id)}
						onChange={(on) => toggle(role.id, on)}
						disabled={why !== undefined}
					>
						{#snippet badges()}
							{#if role.inherited_roles.length > 0}
								<Badge>composite</Badge>
							{/if}
							{#if role.is_default}
								<Badge tone="success">default</Badge>
							{/if}
						{/snippet}
					</Choice>
				{/each}
			</div>
		{:else}
			<p class="empty">{roles.length === 0 ? empty : 'No roles match this.'}</p>
		{/each}
	</div>
</div>

<style>
	.picker {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.list {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		max-height: min(26rem, 55vh);
		padding: var(--space-1);
		overflow-y: auto;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.group {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.heading {
		position: sticky;
		top: calc(var(--space-1) * -1);
		z-index: 1;
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px var(--space-2);
		background: var(--color-surface);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
		font-weight: 700;
		letter-spacing: 0.06em;
		text-transform: uppercase;
	}

	.empty {
		margin: 0;
		padding: var(--space-4) var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		text-align: center;
	}
</style>
