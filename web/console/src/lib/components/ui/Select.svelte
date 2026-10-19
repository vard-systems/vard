<script lang="ts" generics="Value extends string">
	import { createListCollection, Select as ArkSelect } from '@ark-ui/svelte/select';
	import { Portal } from '@ark-ui/svelte/portal';
	import { RiArrowDownSLine, RiCheckLine, RiCloseLine, type IconData } from '#lib/icons.js';
	import FieldText from './FieldText.svelte';
	import Icon from './Icon.svelte';
	import type { SelectOption } from './select';

	type Props = {
		label: string;
		value: Value;
		options: readonly (Value | SelectOption<Value>)[];
		placeholder?: string;
		icon?: IconData;
		hint?: string;
		error?: string;
		required?: boolean;
		disabled?: boolean;
		readOnly?: boolean;
		clearable?: boolean;
		compact?: boolean;
		name?: string;
		id?: string;
		onChange?: (value: Value) => void;
	};

	let {
		label,
		value = $bindable(),
		options,
		placeholder = 'Select…',
		icon,
		hint,
		error,
		required,
		disabled,
		readOnly,
		clearable = false,
		compact = false,
		name,
		id,
		onChange
	}: Props = $props();

	const items = $derived(
		options.map((option) => (typeof option === 'string' ? { value: option } : option))
	);

	const collection = $derived(
		createListCollection({
			items,
			itemToValue: (item) => item.value,
			itemToString: (item) => item.label ?? item.value,
			isItemDisabled: (item) => item.disabled === true
		})
	);

	const chosen = $derived(items.find((item) => item.value === value));

	const gutter = $derived(items.some((item) => item.icon));

	const canClear = $derived(clearable && value !== '' && !disabled && !readOnly);

	let open = $state(false);
</script>

<ArkSelect.Root
	class="field-root"
	data-disabled={disabled || undefined}
	{id}
	{collection}
	{name}
	{required}
	{disabled}
	{readOnly}
	invalid={error !== undefined}
	value={value === '' ? [] : [value]}
	positioning={{ sameWidth: true, gutter: 4 }}
	onOpenChange={(details) => (open = details.open)}
	onValueChange={(details) => {
		value = (details.value[0] ?? '') as Value;
		onChange?.(value);
	}}
>
	<div
		class="field-box"
		data-float={value !== '' || undefined}
		data-open={open || undefined}
		data-compact={compact || undefined}
	>
		<ArkSelect.Label><FieldText {label} {icon} {required} /></ArkSelect.Label>

		<ArkSelect.Control>
			<ArkSelect.Trigger>
				{#if chosen?.icon}
					<Icon icon={chosen.icon} />
				{/if}
				<ArkSelect.ValueText {placeholder} />
				<ArkSelect.Indicator>
					<Icon icon={RiArrowDownSLine} />
				</ArkSelect.Indicator>
			</ArkSelect.Trigger>

			{#if canClear}
				<ArkSelect.ClearTrigger aria-label="Clear {label}">
					<Icon icon={RiCloseLine} />
				</ArkSelect.ClearTrigger>
			{/if}
		</ArkSelect.Control>
	</div>

	<Portal>
		<ArkSelect.Positioner>
			<ArkSelect.Content>
				<ArkSelect.List>
					{#each items as item (item.value)}
						<ArkSelect.Item {item}>
							{#if item.icon}
								<Icon icon={item.icon} />
							{:else if gutter}
								<span data-part="item-gutter"></span>
							{/if}

							<span data-part="item-body">
								<ArkSelect.ItemText>{item.label ?? item.value}</ArkSelect.ItemText>
								{#if item.description}
									<span data-part="item-description">{item.description}</span>
								{/if}
							</span>

							<ArkSelect.ItemIndicator>
								<Icon icon={RiCheckLine} />
							</ArkSelect.ItemIndicator>
						</ArkSelect.Item>
					{:else}
						<p data-part="empty">Nothing to choose from</p>
					{/each}
				</ArkSelect.List>
			</ArkSelect.Content>
		</ArkSelect.Positioner>
	</Portal>

	<ArkSelect.HiddenSelect />

	{#if error}
		<span data-part="error-text">{error}</span>
	{:else if hint}
		<span data-part="helper-text">{hint}</span>
	{/if}
</ArkSelect.Root>
