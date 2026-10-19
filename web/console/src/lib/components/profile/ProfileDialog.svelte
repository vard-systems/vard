<script lang="ts">
	import { RiSettings3Line, RiUserLine } from '#lib/icons.js';
	import type { Admin } from '#lib/api/index.js';
	import { FullscreenDialog, Tabs, Tag, Thumb } from '#lib/components/ui/index.js';
	import { initials } from '#lib/utils/format.js';
	import AccountForms from './AccountForms.svelte';
	import SecuritySettings from './SecuritySettings.svelte';

	type Props = {
		admin: Admin;
		open?: boolean;
	};

	let { admin, open = $bindable(false) }: Props = $props();

	const tabs = [
		{ value: 'account', label: 'Account', icon: RiUserLine },
		{ value: 'settings', label: 'Settings', icon: RiSettings3Line }
	];

	const name = $derived(admin.full_name.trim() || admin.email);

	const standing = $derived(
		admin.is_super_admin
			? 'Super administrator'
			: admin.roles.filter((role) => role !== 'super_admin').join(', ') || 'No roles'
	);
</script>

<FullscreenDialog bind:open title="Your account">
	<div class="summary">
		<Thumb src={admin.avatar_url} text={initials(name)} size="md" shape="circle" tone="accent" />
		<div class="summary-text">
			<strong>{name}</strong>
			<span>{admin.email}</span>
		</div>
		<Tag tone={admin.is_super_admin ? 'info' : 'neutral'} strong>{standing}</Tag>
	</div>

	<Tabs {tabs} value="account" label="Your account">
		{#snippet panel(value)}
			{#if value === 'account'}
				<AccountForms {admin} />
			{:else}
				<SecuritySettings />
			{/if}
		{/snippet}
	</Tabs>
</FullscreenDialog>

<style>
	.summary {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		margin-bottom: var(--space-5);
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.summary-text {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.summary-text strong,
	.summary-text span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.summary-text span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
