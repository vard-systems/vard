<script lang="ts">
	import { LIMITS } from '#lib/constants.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import FieldRow from './FieldRow.svelte';
	import PasswordField from './PasswordField.svelte';

	type Props = {
		password: string;
		confirm: string;
		/** "Password" when there is none yet, "New password" when it replaces one. */
		initial?: boolean;
		/** Side by side, where the page is wide enough for it. */
		row?: boolean;
		disabled?: boolean;
	};

	let {
		password = $bindable(''),
		confirm = $bindable(''),
		initial = false,
		row = false,
		disabled = false
	}: Props = $props();

	const t = useTranslator();

	const mismatch = $derived(confirm !== '' && confirm !== password);
</script>

{#snippet fields()}
	<PasswordField
		label={initial ? t('field.password') : t('field.new_password')}
		bind:value={password}
		autocomplete="new-password"
		{disabled}
		hint={t('field.password_hint', { count: LIMITS.passwordMin })}
	/>
	<PasswordField
		label={initial ? t('field.confirm_password') : t('field.confirm_new_password')}
		bind:value={confirm}
		autocomplete="new-password"
		{disabled}
		error={mismatch ? t('field.password_mismatch') : undefined}
	/>
{/snippet}

{#if row}
	<FieldRow>{@render fields()}</FieldRow>
{:else}
	{@render fields()}
{/if}
