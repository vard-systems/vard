<script lang="ts">
	import { useQueryClient } from '@tanstack/svelte-query';
	import { usersApi, type UserRecord } from '#lib/api/index.js';
	import {
		Button,
		FormSection,
		List,
		ListItem,
		Tag,
		Thumb,
		notify
	} from '#lib/components/ui/index.js';
	import { markFor as providerMark } from '#lib/components/social/providers.js';
	import { keys } from '#lib/query/index.js';
	import { formatDate, formatRelative } from '#lib/utils/format.js';

	type Props = {
		user: UserRecord;
		editable: boolean;
		/** Called with what is left once one is disconnected. */
		ondisconnected: (user: UserRecord) => void;
	};

	let { user, editable, ondisconnected }: Props = $props();

	const queryClient = useQueryClient();

	let disconnecting = $state('');

	async function disconnect(identity: string) {
		disconnecting = identity;

		try {
			await usersApi.disconnect(user.id, identity);
			notify.success('Provider disconnected');
			await queryClient.invalidateQueries({ queryKey: keys.users.all });

			ondisconnected({
				...user,
				social_accounts: (user.social_accounts ?? []).filter((account) => account.id !== identity)
			});
		} catch (err) {
			notify.error(err, 'Could not disconnect this provider');
		} finally {
			disconnecting = '';
		}
	}
</script>

<FormSection
	title="Signs in with"
	description="Accounts elsewhere that reach this one. Disconnecting one leaves the account itself alone."
>
	<List bordered label="Connected providers">
		{#each user.social_accounts ?? [] as account (account.id)}
			<ListItem
				title={account.provider}
				description={account.email || `Connected ${formatDate(account.connected_at)}`}
			>
				{#snippet lead()}
					<Thumb icon={providerMark(account.kind)} />
				{/snippet}

				{#snippet end()}
					{#if account.last_login_at}
						<Tag small>used {formatRelative(account.last_login_at)}</Tag>
					{/if}

					{#if editable}
						<Button
							size="sm"
							variant="subtle"
							colorPalette="danger"
							loading={disconnecting === account.id}
							disabled={disconnecting !== ''}
							onclick={() => disconnect(account.id)}
						>
							Disconnect
						</Button>
					{/if}
				{/snippet}
			</ListItem>
		{/each}
	</List>
</FormSection>
