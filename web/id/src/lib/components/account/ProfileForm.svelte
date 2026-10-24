<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { account, type User } from '#lib/api/index.js';
	import { LIMITS } from '#lib/constants.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { Button, FieldRow, Form, Panel, TextField } from '../ui/index.js';
	import EmailChange from './EmailChange.svelte';

	type Props = {
		user: User;
		/** Whether the login flow lets people change their address themselves. */
		mayChangeEmail: boolean;
	};

	let { user, mayChangeEmail }: Props = $props();

	const t = useTranslator();

	// The form starts from the saved names and is the reader's from then on.
	// svelte-ignore state_referenced_locally
	let firstName = $state(user.first_name);
	// svelte-ignore state_referenced_locally
	let lastName = $state(user.last_name);
	let saving = $state(false);

	const changed = $derived(
		firstName.trim() !== user.first_name || lastName.trim() !== user.last_name
	);

	function reset() {
		firstName = user.first_name;
		lastName = user.last_name;
	}

	async function save() {
		if (!changed || saving) return;

		saving = true;

		try {
			const { user: updated } = await account.updateProfile({
				first_name: firstName.trim(),
				last_name: lastName.trim()
			});
			firstName = updated.first_name;
			lastName = updated.last_name;
			notify.success(t('profile.saved'));
			await refreshAll();
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			saving = false;
		}
	}
</script>

<Form onsubmit={save}>
	<Panel title={t('profile.personal_title')} description={t('profile.personal_description')}>
		<FieldRow>
			<TextField
				label={t('field.first_name')}
				bind:value={firstName}
				autocomplete="given-name"
				maxlength={LIMITS.name}
				disabled={saving}
			/>
			<TextField
				label={t('field.last_name')}
				bind:value={lastName}
				autocomplete="family-name"
				maxlength={LIMITS.name}
				disabled={saving}
			/>
		</FieldRow>

		<TextField
			label={t('field.email')}
			value={user.email}
			readonly
			hint={mayChangeEmail ? t('account.email_change_body') : t('profile.email_hint')}
		/>

		{#if mayChangeEmail}
			<EmailChange />
		{/if}

		{#snippet footer()}
			<Button variant="secondary" disabled={!changed || saving} onclick={reset}>
				{t('action.cancel')}
			</Button>
			<Button type="submit" loading={saving} disabled={!changed}>
				{t('action.save_changes')}
			</Button>
		{/snippet}
	</Panel>
</Form>
