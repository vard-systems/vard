<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { account, type Factor } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { formatDate, timeAgo } from '#lib/utils/format.js';
	import { Button, ItemRow, type IconName } from '../ui/index.js';
	import PresenceCheck from './PresenceCheck.svelte';

	type Props = {
		factor: Factor;
		icon: IconName;
		name: string;
		asksPassword: boolean;
		timezone: string;
	};

	let { factor, icon, name, asksPassword, timezone }: Props = $props();

	const t = useTranslator();

	let confirming = $state(false);
	let removing = $state(false);

	const detail = $derived(
		t('security.factor_line', {
			added: formatDate(factor.added_at, t, timezone),
			used: factor.last_used_at ? timeAgo(factor.last_used_at, t) : t('time.never')
		})
	);

	async function remove(password: string) {
		removing = true;

		try {
			await account.removeFactor(factor.id, password);
			notify.success(t('security.removed'));
			await refreshAll();
		} catch (err) {
			notify.error(messageOf(err, t));
			removing = false;
		}
	}
</script>

{#snippet check()}
	<PresenceCheck
		{asksPassword}
		confirm={t('security.remove')}
		danger
		busy={removing}
		onconfirm={remove}
		oncancel={() => (confirming = false)}
	/>
{/snippet}

<ItemRow {icon} {detail} below={confirming ? check : undefined}>
	{#snippet title()}{name}{/snippet}
	{#snippet actions()}
		{#if !confirming}
			<Button variant="danger" size="sm" onclick={() => (confirming = true)}>
				{t('security.remove')}
			</Button>
		{/if}
	{/snippet}
</ItemRow>
