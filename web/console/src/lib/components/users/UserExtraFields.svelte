<script lang="ts">
	import type { UserField } from '#lib/api/index.js';
	import { FormSection } from '#lib/components/ui/index.js';
	import FieldInput from './FieldInput.svelte';
	import type { FieldValues } from './form.svelte';

	type Props = { extras: UserField[]; values: FieldValues };

	let { extras, values = $bindable({}) }: Props = $props();

	/** Text-like fields one under another, then the checkboxes side by side. */
	const details = $derived(extras.filter((field) => field.type !== 'bool'));
	const flags = $derived(extras.filter((field) => field.type === 'bool'));
</script>

<FormSection
	title="Additional fields"
	description="The fields this organisation keeps about its users."
>
	{#each details as field (field.id)}
		<FieldInput {field} bind:value={values[field.name]} />
	{/each}

	{#if flags.length > 0}
		<div class="flags">
			{#each flags as field (field.id)}
				<FieldInput {field} bind:value={values[field.name]} />
			{/each}
		</div>
	{/if}
</FormSection>

<style>
	.flags {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: var(--space-2) var(--space-4);
	}
</style>
