<script lang="ts">
	import { RiAddLine, RiDeleteBinLine } from 'svelte-remixicon';
	import type { Role, SSORoleMapping } from '$lib/api';
	import { Button, Icon, IconButton, Input, Note, Select } from '$lib/components/ui';
	import { resolve } from '$app/paths';

	type Props = {
		mappings: SSORoleMapping[];
		roles: Role[];
		canRead: boolean;
		readOnly?: boolean;
	};

	let { mappings = $bindable([]), roles, canRead, readOnly = false }: Props = $props();

	const options = $derived(
		roles.map((role) => ({
			value: role.id,
			label: role.name,
			description: role.application_id ? 'Application role' : 'Global role'
		}))
	);

	function add() {
		mappings = [...mappings, { group: '', role_id: roles[0]?.id ?? '' }];
	}

	function remove(index: number) {
		mappings = mappings.filter((_, i) => i !== index);
	}
</script>

<div class="mappings">
	{#each mappings as mapping, index (index)}
		<div class="row">
			<Input
				label="Group"
				bind:value={mapping.group}
				placeholder="engineering"
				{readOnly}
				autocomplete="off"
			/>
			{#if roles.length > 0}
				<Select label="Role" bind:value={mapping.role_id} {options} {readOnly} />
			{:else}
				<Input label="Role" value={mapping.role_id} readOnly />
			{/if}
			{#if !readOnly}
				<IconButton icon={RiDeleteBinLine} label="Remove mapping" onclick={() => remove(index)} />
			{/if}
		</div>
	{/each}

	{#if roles.length === 0 && !readOnly && !canRead}
		<Note>Your roles do not let you read roles, so you cannot map groups to them.</Note>
	{:else if roles.length === 0 && !readOnly}
		<Note>
			There are no roles yet, so there is nothing to map a group to.
			<a href={resolve('/admin/dashboard/roles')}>Create one on the Roles page</a>
		</Note>
	{:else if !readOnly}
		<div>
			<Button variant="subtle" size="sm" onclick={add}>
				<Icon icon={RiAddLine} />
				Add mapping
			</Button>
		</div>
	{/if}
</div>

<style>
	.mappings {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.row {
		display: grid;
		grid-template-columns: 1fr 1fr auto;
		align-items: center;
		gap: var(--space-2);
	}

	@media (max-width: 36rem) {
		.row {
			grid-template-columns: 1fr;
		}
	}
</style>
