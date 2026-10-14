<script lang="ts">
	import { createMutation } from '@tanstack/svelte-query';
	import { RiShieldKeyholeLine } from 'svelte-remixicon';
	import { adminsApi, type AdminSecurity } from '#lib/api/index.js';
	import {
		Alert,
		Panel,
		SwitchField,
		Tag,
		ConfirmDialog,
		notify
	} from '#lib/components/ui/index.js';

	type Props = {
		security: AdminSecurity;
		selfHasMFA: boolean;
	};

	let { security, selfHasMFA }: Props = $props();

	// svelte-ignore state_referenced_locally
	let current = $state(security);

	let saving = $state(false);
	let confirming = $state(false);

	/** How many administrators have nothing to sign in with yet. A change that
	    would shut them out of everything but the setup page is worth saying
	    out loud before it is made. */
	const without = $derived(Math.max(current.administrators - current.with_mfa, 0));

	const save = createMutation(() => ({
		mutationFn: (required: boolean) => adminsApi.updateSecurity({ require_mfa: required }),
		onSuccess: (result) => {
			notify.success(
				result.require_mfa
					? 'Administrators now need a second factor'
					: 'Administrators no longer need a second factor'
			);
			current = result;
			confirming = false;
		},
		onError: (err: unknown) => {
			confirming = false;
			notify.error(err, 'Could not change this setting');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	function set(required: boolean) {
		if (required && !confirming) {
			confirming = true;
			return;
		}

		saving = true;
		save.mutate(required);
	}
</script>

<Panel title="Two-factor sign-in" icon={RiShieldKeyholeLine}>
	{#snippet meta()}
		<Tag tone={without === 0 ? 'info' : 'danger'} small>
			{current.with_mfa} of {current.administrators}
			{current.administrators === 1 ? 'administrator has one' : 'administrators have one'}
		</Tag>
	{/snippet}

	<SwitchField
		label="Require an authenticator"
		description="Every administrator sets one up before they can do anything else, and none of them can turn it off again."
		checked={current.require_mfa}
		disabled={saving}
		onChange={(on) => set(on)}
	/>
</Panel>

<ConfirmDialog
	bind:open={confirming}
	title="Require two-factor sign-in?"
	description={without > 0
		? `${without} ${without === 1 ? 'administrator has' : 'administrators have'} no authenticator yet. They will be able to do nothing but set one up — including you, if that is you.`
		: 'Every administrator already has one, so nothing changes for them today. New ones will be asked to set one up before anything else.'}
	tone="danger"
	icon={RiShieldKeyholeLine}
	confirmLabel="Require it"
	busy={saving}
	onConfirm={() => set(true)}
>
	{#if !selfHasMFA}
		<Alert tone="danger">
			<strong>You have none yet:</strong> the next page you load will ask you to set one up, and you will
			not be able to turn this off again until you have.
		</Alert>
	{/if}
</ConfirmDialog>

<style>
</style>
