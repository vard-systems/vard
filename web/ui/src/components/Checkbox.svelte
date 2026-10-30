<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Checkbox } from '@ark-ui/svelte/checkbox';

	type Props = {
		checked: boolean | 'indeterminate';
		onChange: (checked: boolean) => void;
		label?: string;
		/** A label with more in it than words (a link to the terms); `title` then names the box. */
		children?: Snippet;
		title?: string;
		disabled?: boolean;
		id?: string;
		name?: string;
	};

	let { checked, onChange, label, children, title, disabled, id, name }: Props = $props();
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
			<i class="ri-check-line" style:font-size="14px" aria-hidden="true"></i>
		</Checkbox.Indicator>
		<Checkbox.Indicator indeterminate>
			<i class="ri-subtract-line" style:font-size="14px" aria-hidden="true"></i>
		</Checkbox.Indicator>
	</Checkbox.Control>

	{#if children}
		<Checkbox.Label>{@render children()}</Checkbox.Label>
	{:else if label}
		<Checkbox.Label>{label}</Checkbox.Label>
	{/if}

	<Checkbox.HiddenInput aria-label={label ?? title} />
</Checkbox.Root>
