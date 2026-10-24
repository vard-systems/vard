<script module lang="ts">
	import type { ApplicationType } from '#lib/api/index.js';
	import type { SelectOption } from '#lib/components/ui/index.js';
	import { types } from '../applications';

	const typeOptions: SelectOption<ApplicationType>[] = (
		Object.entries(types) as [ApplicationType, (typeof types)[ApplicationType]][]
	).map(([value, meta]) => ({
		value,
		label: meta.label,
		description: meta.description,
		icon: meta.icon
	}));
</script>

<script lang="ts">
	import type { Application } from '#lib/api/index.js';
	import {
		FieldGrid,
		FormSection,
		Input,
		Select,
		SwitchField,
		Textarea
	} from '#lib/components/ui/index.js';
	import type { ApplicationForm } from '../form.svelte';

	type Props = { form: ApplicationForm; current: Application | null };

	let { form, current }: Props = $props();
</script>

<FormSection title="General" description="How the application is named and presented.">
	<Select
		label="Application type"
		value={form.input.type}
		options={typeOptions}
		readOnly={current !== null}
		hint={current
			? 'The type cannot change: copies of the app already in use depend on it.'
			: form.kind.description}
		onChange={(type) => form.changeType(type)}
	/>

	<Input label="Name" bind:value={form.input.name} placeholder="Shop" required />

	<Textarea
		label="Description"
		bind:value={form.input.description}
		rows={2}
		placeholder="What the application is, for other administrators"
	/>

	<FieldGrid>
		<Input
			label="Homepage URL"
			bind:value={form.input.website_url}
			type="url"
			placeholder="https://shop.example.com"
		/>
		<Input
			label="Logo URL"
			bind:value={form.input.logo_url}
			type="url"
			placeholder="https://shop.example.com/logo.png"
			hint="https only."
		/>
	</FieldGrid>

	<SwitchField
		label="Is Enabled"
		description="A disabled application cannot sign anyone in."
		bind:checked={form.input.is_enabled}
	/>
</FormSection>
