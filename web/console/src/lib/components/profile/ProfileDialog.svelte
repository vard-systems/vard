<script lang="ts">
	import type { Admin } from '#lib/api/index.js';
	import { FullscreenDialog, Tabs, Thumb } from '#lib/components/ui/index.js';
	import { initials } from '#lib/utils/format.js';
	import AccountForms from './AccountForms.svelte';
	import RoleCodes from './RoleCodes.svelte';
	import SecuritySettings from './SecuritySettings.svelte';

	type Props = {
		admin: Admin;
		open?: boolean;
	};

	let { admin, open = $bindable(false) }: Props = $props();

	const tabs = [
		{ value: 'account', label: 'Account', icon: 'ri-user-line' },
		{ value: 'settings', label: 'Settings', icon: 'ri-settings-3-line' }
	];

	const name = $derived(admin.full_name.trim() || admin.email);
</script>

<FullscreenDialog bind:open title="Your account">
	<section class="summary" aria-label="Signed in as">
		<Thumb src={admin.avatar_url} text={initials(name)} size="md" shape="circle" tone="accent" />
		<div class="who">
			<strong title={name}>{name}</strong>
			{#if admin.email !== name}<span title={admin.email}>{admin.email}</span>{/if}
		</div>
		<div class="roles"><RoleCodes {admin} /></div>
	</section>

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
	/* The avatar, the person, then their roles; on a narrow screen the roles
	   drop under the name rather than squeezing it. */
	.summary {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr) auto;
		align-items: center;
		gap: var(--space-2) var(--space-3);
		margin-bottom: var(--space-5);
		padding: var(--space-3) var(--space-4);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.who {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
		line-height: 1.3;
	}

	.who strong,
	.who span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.who span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.roles {
		justify-self: end;
		max-width: 20rem;
	}

	@media (max-width: 40rem) {
		.summary {
			grid-template-columns: auto minmax(0, 1fr);
		}

		.roles {
			grid-column: 2;
			justify-self: start;
			max-width: none;
		}
	}
</style>
