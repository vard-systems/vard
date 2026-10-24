<script lang="ts">
	import { useQueryClient } from '@tanstack/svelte-query';
	import { usersApi, type UserRecord } from '#lib/api/index.js';
	import {
		Button,
		ConfirmDialog,
		FormSection,
		List,
		ListItem,
		Thumb,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';

	type Props = {
		user: UserRecord;
		editable: boolean;
		/** Called once it is off, so the dialog can show it. */
		onreset: () => void;
	};

	let { user, editable, onreset }: Props = $props();

	const queryClient = useQueryClient();

	let confirming = $state(false);
	let resetting = $state(false);

	async function reset() {
		resetting = true;

		try {
			await usersApi.resetTwoStep(user.id);
			notify.success('Two-step verification reset');
			confirming = false;
			onreset();
			await queryClient.invalidateQueries({ queryKey: keys.users.all });
		} catch (err) {
			notify.error(err, 'Could not reset two-step verification');
		} finally {
			resetting = false;
		}
	}
</script>

<FormSection
	title="Two-step verification"
	description="The user turned it on themselves. Reset it only for someone who lost every device and recovery code."
>
	<List bordered label="Two-step verification">
		<ListItem
			title="On"
			description="Asked for an authenticator code or a passkey at every sign-in."
		>
			{#snippet lead()}
				<Thumb icon="ri-shield-keyhole-line" />
			{/snippet}

			{#snippet end()}
				{#if editable}
					<Button
						size="sm"
						variant="subtle"
						colorPalette="danger"
						onclick={() => (confirming = true)}
					>
						Reset
					</Button>
				{/if}
			{/snippet}
		</ListItem>
	</List>
</FormSection>

<ConfirmDialog
	bind:open={confirming}
	title="Reset two-step verification?"
	description={`${user.email} will sign in with their password or other account alone, until they turn it on again. Their authenticator, passkeys and recovery codes are removed.`}
	icon="ri-shield-cross-line"
	confirmLabel="Reset"
	busy={resetting}
	onConfirm={reset}
/>
