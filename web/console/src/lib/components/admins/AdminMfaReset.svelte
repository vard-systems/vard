<script lang="ts">
	import { useQueryClient } from '@tanstack/svelte-query';
	import { adminsApi, type AdminRecord } from '#lib/api/index.js';
	import { Badge, Button, ConfirmDialog, FormSection, notify } from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';

	type Props = {
		admin: AdminRecord;
		/** Called once it is reset, so the dialog can close. */
		onreset: () => void;
	};

	let { admin, onreset }: Props = $props();

	const queryClient = useQueryClient();

	let confirming = $state(false);
	let resetting = $state(false);

	async function reset() {
		resetting = true;

		try {
			await adminsApi.resetMfa(admin.id);
			notify.success(
				'Two-factor sign-in reset',
				'They set up a new authenticator at their next sign-in.'
			);
			await queryClient.invalidateQueries({ queryKey: keys.admins.all });
			onreset();
		} catch (err) {
			notify.error(err, 'Could not reset two-factor sign-in');
		} finally {
			resetting = false;
			confirming = false;
		}
	}
</script>

<FormSection
	title="Two-factor sign-in"
	description="Whether they sign in with a code from an authenticator app."
>
	<div class="mfa">
		<Badge>{admin.mfa_enabled ? 'On' : 'Off'}</Badge>
		{#if admin.mfa_enabled}
			<Button size="sm" variant="subtle" onclick={() => (confirming = true)}>
				Reset two-factor
			</Button>
		{/if}
	</div>
	<p class="hint">
		For someone who lost their phone and their recovery codes. They are signed out, and set it up
		again at their next sign-in where it is required.
	</p>
</FormSection>

<ConfirmDialog
	bind:open={confirming}
	title="Reset two-factor sign-in?"
	description={`${admin.email} loses their authenticator and recovery codes, and is signed out everywhere. They set it up again at their next sign-in where it is required.`}
	tone="warning"
	icon="ri-shield-keyhole-line"
	confirmLabel="Reset two-factor"
	busy={resetting}
	onConfirm={reset}
/>

<style>
	.mfa {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
	}

	.hint {
		margin: calc(var(--space-2) * -1) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.4;
	}
</style>
