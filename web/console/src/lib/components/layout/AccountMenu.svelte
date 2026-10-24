<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { adminApi, type Admin } from '#lib/api/index.js';
	import {
		ConfirmDialog,
		DropdownMenu,
		MenuItem,
		MenuSeparator,
		Thumb
	} from '#lib/components/ui/index.js';
	import { initials } from '#lib/utils/format.js';
	import ProfileDialog from '#lib/components/profile/ProfileDialog.svelte';
	import RoleCodes from '#lib/components/profile/RoleCodes.svelte';

	type Props = { admin: Admin };

	let { admin }: Props = $props();

	const queryClient = useQueryClient();

	let signingOut = $state(false);
	let accountOpen = $state(false);
	let confirmingSignOut = $state(false);

	const name = $derived(admin.full_name.trim() || admin.email);

	const monogram = $derived(initials(name));

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
			await goto(resolve('/login'), { replace: true });
		}
	}
</script>

<DropdownMenu label="Account: {name}" shape="labelled" width="16rem">
	{#snippet trigger()}
		<Thumb src={admin.avatar_url} text={monogram} size="xs" shape="circle" tone="accent" />
		<span class="name">{name}</span>
		<span class="chevron" aria-hidden="true"
			><i class="ri-arrow-down-s-line" aria-hidden="true"></i></span
		>
	{/snippet}

	<div class="card">
		<Thumb src={admin.avatar_url} text={monogram} size="sm" shape="circle" tone="accent" />
		<div class="who">
			<span class="full-name" title={name}>{name}</span>
			{#if admin.email !== name}<span class="email" title={admin.email}>{admin.email}</span>{/if}
		</div>

		<div class="roles"><RoleCodes {admin} /></div>
	</div>

	<MenuSeparator />

	<MenuItem
		value="account"
		icon="ri-user-settings-line"
		label="Account settings"
		onSelect={() => (accountOpen = true)}
	/>

	<MenuSeparator />

	<MenuItem
		value="sign-out"
		icon="ri-logout-box-r-line"
		label="Sign out"
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
	icon="ri-logout-box-r-line"
	confirmLabel="Sign out"
	busy={signingOut}
	onConfirm={signOut}
/>

<style>
	/* Two columns: the avatar, then everything about the person, so the
	   name, address and roles start on one line. */
	.card {
		display: grid;
		grid-template-columns: auto minmax(0, 1fr);
		align-items: center;
		gap: var(--space-2) 10px;
		padding: 10px;
	}

	.who {
		display: flex;
		flex-direction: column;
		min-width: 0;
		line-height: 1.3;
	}

	.full-name,
	.email {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.full-name {
		font-weight: 600;
	}

	.email {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.roles {
		grid-column: 2;
	}

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
