<script lang="ts">
	import { useTranslator } from '#lib/i18n/index.js';
	import { Button, Form, PasswordField } from '../ui/index.js';

	type Props = {
		/** An account without a password is not asked for one. */
		asksPassword: boolean;
		confirm: string;
		danger?: boolean;
		busy?: boolean;
		onconfirm: (password: string) => unknown;
		oncancel: () => void;
	};

	let {
		asksPassword,
		confirm,
		danger = false,
		busy = false,
		onconfirm,
		oncancel
	}: Props = $props();

	const t = useTranslator();

	let password = $state('');
</script>

<Form onsubmit={() => onconfirm(password)}>
	{#if asksPassword}
		<PasswordField label={t('security.confirm_password')} bind:value={password} disabled={busy} />
	{/if}
	<div class="actions">
		<Button variant="ghost" size="sm" onclick={oncancel} disabled={busy}>
			{t('action.cancel')}
		</Button>
		<Button
			type="submit"
			variant={danger ? 'danger' : 'primary'}
			size="sm"
			loading={busy}
			disabled={asksPassword && password === ''}
		>
			{confirm}
		</Button>
	</div>
</Form>

<style>
	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
