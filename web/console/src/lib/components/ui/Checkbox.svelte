<script lang="ts">
	import { Checkbox } from '@ark-ui/svelte/checkbox';
	import { RiCheckLine, RiSubtractLine } from 'svelte-remixicon';
	import Icon from './Icon.svelte';

	type Props = {
		checked: boolean | 'indeterminate';
		onChange: (checked: boolean) => void;
		label?: string;
		title?: string;
		disabled?: boolean;
		id?: string;
		name?: string;
	};

	let { checked, onChange, label, title, disabled, id, name }: Props = $props();
</script>

<Checkbox.Root
	{id}
	{name}
	{checked}
	{disabled}
	onCheckedChange={(details) => onChange(details.checked === true)}
>
	<Checkbox.Control>
		<Checkbox.Indicator>
			<Icon icon={RiCheckLine} size="0.875rem" />
		</Checkbox.Indicator>
		<Checkbox.Indicator indeterminate>
			<Icon icon={RiSubtractLine} size="0.875rem" />
		</Checkbox.Indicator>
	</Checkbox.Control>

	{#if label}
		<Checkbox.Label>{label}</Checkbox.Label>
	{/if}

	<Checkbox.HiddenInput aria-label={label ?? title} />
</Checkbox.Root>
