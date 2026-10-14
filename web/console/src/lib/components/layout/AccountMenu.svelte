<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { useQueryClient } from '@tanstack/svelte-query';
	import {
		RiArrowDownSLine,
		RiLogoutBoxRLine,
		RiShieldUserLine,
		RiUserSettingsLine
	} from 'svelte-remixicon';
	import { adminApi, type Admin } from '#lib/api/index.js';
	import {
		ConfirmDialog,
		DropdownMenu,
		Icon,
		MenuGroup,
		MenuInfo,
		MenuItem,
		MenuSeparator,
		Thumb
	} from '#lib/components/ui/index.js';
	import { initials } from '#lib/utils/format.js';
	import ProfileDialog from '#lib/components/profile/ProfileDialog.svelte';

	type Props = { admin: Admin };

	let { admin }: Props = $props();

	const queryClient = useQueryClient();

	let signingOut = $state(false);
	let accountOpen = $state(false);
	let confirmingSignOut = $state(false);

	const name = $derived(admin.full_name.trim() || admin.email);

	const monogram = $derived(initials(name));

	const role = $derived(
		[
			...(admin.is_super_admin ? ['Super administrator'] : []),
			...admin.roles.filter((name) => name !== 'super_admin')
		].join(', ') || 'No roles'
	);

	async function signOut() {
		signingOut = true;

		try {
			await adminApi.logout();
		} finally {
			// However the server answered, this browser is done with the
			// session: drop everything that was loaded with it — the pages
			// and the cache behind them — and go to the sign-in page. What
			// one administrator saw is not for whoever signs in next.
			queryClient.clear();
			await refreshAll();
			await goto(resolve('admin/login'), { replace: true });
		}
	}
</script>

<DropdownMenu label="Account: {name}" shape="labelled">
	{#snippet trigger()}
		<Thumb src={admin.avatar_url} text={monogram} size="xs" shape="circle" tone="accent" />
		<span class="name">{name}</span>
		<span class="chevron" aria-hidden="true"><Icon icon={RiArrowDownSLine} size="1rem" /></span>
	{/snippet}

	<MenuGroup label="Signed in as">
		<MenuInfo label={name} description={admin.email}>
			{#snippet lead()}
				<Thumb src={admin.avatar_url} text={monogram} size="xs" shape="circle" tone="accent" />
			{/snippet}
		</MenuInfo>
		<MenuInfo icon={RiShieldUserLine} label={role} description="Role" />
	</MenuGroup>

	<MenuSeparator />

	<MenuGroup label="Account">
		<MenuItem
			value="account"
			icon={RiUserSettingsLine}
			label="Account settings"
			description="Profile, security and sessions"
			onSelect={() => (accountOpen = true)}
		/>
	</MenuGroup>

	<MenuSeparator />

	<MenuItem
		value="sign-out"
		icon={RiLogoutBoxRLine}
		label="Sign out"
		description="End this session"
		danger
		onSelect={() => (confirmingSignOut = true)}
	/>
</DropdownMenu>

<ProfileDialog {admin} bind:open={accountOpen} />

<ConfirmDialog
	bind:open={confirmingSignOut}
	title="Sign out?"
	description="You will need to sign in again to manage the console."
	tone="danger"
	icon={RiLogoutBoxRLine}
	confirmLabel="Sign out"
	busy={signingOut}
	onConfirm={signOut}
/>

<style>
	.name {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.chevron {
		display: inline-flex;
		flex: none;
		color: var(--color-text-hint);
		transition: transform var(--speed);
	}

	:global(.dropdown-trigger[data-state='open']) .chevron {
		transform: rotate(180deg);
	}

	@media (max-width: 64rem) {
		.name,
		.chevron {
			display: none;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.chevron {
			transition: none;
		}
	}
</style>
