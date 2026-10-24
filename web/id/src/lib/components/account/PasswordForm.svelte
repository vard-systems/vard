<script lang="ts">
	import { refreshAll } from '$app/navigation';
	import { account } from '#lib/api/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { notify } from '#lib/state/toast.svelte.js';
	import { newPasswordReady } from '#lib/utils/password.js';
	import { Button, Form, NewPasswordFields, Panel, PasswordField } from '../ui/index.js';

	const t = useTranslator();

	let current = $state('');
	let password = $state('');
	let confirm = $state('');
	let changing = $state(false);

	const canChange = $derived(current !== '' && newPasswordReady(password, confirm) && !changing);

	async function change() {
		if (!canChange) return;

		changing = true;

		try {
			await account.changePassword({ current_password: current, new_password: password });
			current = password = confirm = '';
			notify.success(t('security.password_changed'));
			await refreshAll();
		} catch (err) {
			notify.error(messageOf(err, t));
		} finally {
			changing = false;
		}
	}
</script>

<Form onsubmit={change}>
	<Panel title={t('security.password_title')} description={t('security.password_description')}>
		<PasswordField label={t('field.current_password')} bind:value={current} disabled={changing} />
		<NewPasswordFields bind:password bind:confirm row disabled={changing} />

		{#snippet footer()}
			<Button type="submit" loading={changing} disabled={!canChange}>
				{t('security.password_title')}
			</Button>
		{/snippet}
	</Panel>
</Form>
