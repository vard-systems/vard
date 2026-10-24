<script lang="ts">
	import type { Admin } from '#lib/api/index.js';
	import { Code } from '#lib/components/ui/index.js';

	type Props = { admin: Admin };

	let { admin }: Props = $props();

	/** The panel-wide roles by the names they have in code, super_admin first. */
	const roles = $derived(
		[...admin.roles].sort((a, b) => Number(b === 'super_admin') - Number(a === 'super_admin'))
	);

	/** Roles held only in some applications are not in admin.roles. */
	const scopedOnly = $derived(
		roles.length === 0 && Object.keys(admin.scoped_permissions).length > 0
	);
</script>

<ul class="roles" aria-label="Roles">
	{#each roles as role (role)}
		<li><Code>{role}</Code></li>
	{:else}
		<li class="none">{scopedOnly ? 'Roles in some applications' : 'No roles'}</li>
	{/each}
</ul>

<style>
	.roles {
		display: flex;
		flex-wrap: wrap;
		gap: 4px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.none {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}
</style>
