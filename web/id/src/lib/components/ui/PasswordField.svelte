<script lang="ts">
	import { PasswordInput } from 'ui';
	import { useTranslator } from '#lib/i18n/index.js';

	type Props = {
		label: string;
		value: string;
		autocomplete?: 'current-password' | 'new-password';
		disabled?: boolean;
		hint?: string;
		error?: string;
		aside?: { label: string; href: string };
	};

	let {
		label,
		value = $bindable(''),
		autocomplete = 'current-password',
		disabled = false,
		hint,
		error,
		aside
	}: Props = $props();

	const t = useTranslator();
</script>

<div class="password">
	<PasswordInput
		{label}
		bind:value
		{autocomplete}
		{disabled}
		{hint}
		{error}
		revealLabel={t('field.show_password')}
	/>
	{#if aside}
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- built with resolve() by the page -->
		<a class="aside" href={aside.href}>{aside.label}</a>
	{/if}
</div>

<style>
	.password {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.aside {
		align-self: flex-end;
		font-size: var(--text-sm);
	}
</style>
