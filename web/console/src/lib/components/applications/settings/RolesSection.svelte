<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Application } from '#lib/api/index.js';
	import { FormSection, SwitchField } from '#lib/components/ui/index.js';
	import type { ApplicationForm } from '../form.svelte';

	type Props = { form: ApplicationForm; current: Application | null };

	let { form, current }: Props = $props();

	const rolesHref = $derived(
		current ? `${resolve('/(panel)/roles')}?tab=application&application=${current.id}` : ''
	);
</script>

<FormSection title="Roles" description="How the application's roles reach its tokens.">
	{#snippet action()}
		{#if current}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a class="link" href={rolesHref}>
				Manage {current.role_count}
				{current.role_count === 1 ? 'role' : 'roles'}
				<i class="ri-arrow-right-line" style:font-size="14px" aria-hidden="true"></i>
			</a>
		{/if}
	{/snippet}

	<div class="switches">
		<SwitchField
			label="Include roles in tokens"
			description="Put the user's roles in this application into its tokens and userinfo."
			bind:checked={form.input.assert_roles}
		/>
		<SwitchField
			label="Require a role to sign in"
			description="Only let users who hold at least one of its roles sign in to it."
			bind:checked={form.input.require_role_assignment}
		/>
	</div>
</FormSection>

<style>
	.switches {
		display: flex;
		flex-direction: column;
	}

	.link {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		color: var(--color-text);
		font-size: var(--text-sm);
		font-weight: 600;
		text-decoration: none;
		white-space: nowrap;
	}

	.link:hover {
		text-decoration: underline;
	}
</style>
