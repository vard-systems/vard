<script lang="ts">
	import type { UserField } from '$lib/api';
	import { DatePicker, Input, Switch } from '$lib/components/ui';
	import { fieldIcons } from './fieldIcons';

	type Props = {
		field: UserField;
		value: string | boolean | undefined;
	};

	let { field, value = $bindable() }: Props = $props();

	const inputType = $derived(field.type === 'number' ? 'number' : 'text');
</script>

{#if field.type === 'bool'}
	<Switch label={field.label} checked={value === true} onChange={(on) => (value = on)} />
{:else if field.type === 'date'}
	<DatePicker
		label={field.label}
		icon={fieldIcons.date}
		value={String(value ?? '')}
		onChange={(day) => (value = day)}
		required={field.is_required}
		clearable={!field.is_required}
	/>
{:else}
	<Input
		label={field.label}
		icon={fieldIcons[field.type]}
		value={String(value ?? '')}
		oninput={(event) => (value = event.currentTarget.value)}
		type={inputType}
		required={field.is_required}
	/>
{/if}
