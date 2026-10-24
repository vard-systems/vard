<script lang="ts">
	import type { Admin, Application, Role } from '#lib/api/index.js';
	import { Button, IconButton } from '#lib/components/ui/index.js';
	import RolePicker from './RolePicker.svelte';
	import { canAssign } from './roles';

	type Props = {
		/** The roles the user does not hold yet. */
		roles: Role[];
		applications: Application[];
		admin: Admin;
		busy: boolean;
		onassign: (ids: string[]) => void;
		oncancel: () => void;
	};

	let { roles, applications, admin, busy, onassign, oncancel }: Props = $props();

	let picked = $state<string[]>([]);
</script>

<div class="head">
	<IconButton
		icon="ri-arrow-left-line"
		label="Back to the user's roles"
		size="sm"
		onclick={oncancel}
		disabled={busy}
	/>
	<div>
		<strong>Assign roles</strong>
		<p>Pick any number of roles, global or from any application.</p>
	</div>
</div>

<RolePicker
	{roles}
	{applications}
	bind:picked
	reason={(role) =>
		canAssign(admin, role) ? undefined : 'Your roles do not allow assigning this role.'}
	empty="This user already has every role there is."
/>

<div class="foot">
	<span class="hint">
		{picked.length === 0
			? 'Nothing selected'
			: `${picked.length} ${picked.length === 1 ? 'role' : 'roles'} selected`}
	</span>
	<Button variant="subtle" size="sm" onclick={oncancel} disabled={busy}>Cancel</Button>
	<Button
		size="sm"
		onclick={() => onassign(picked)}
		disabled={busy || picked.length === 0}
		loading={busy}
	>
		Assign{picked.length > 0 ? ` ${picked.length}` : ''}
	</Button>
</div>

<style>
	.head {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.head p {
		margin: 2px 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.foot {
		display: flex;
		align-items: center;
		justify-content: flex-end;
		gap: var(--space-2);
		margin-top: var(--space-4);
		padding-top: var(--space-3);
		border-top: 1px solid var(--color-border);
	}

	.hint {
		flex: 1;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
