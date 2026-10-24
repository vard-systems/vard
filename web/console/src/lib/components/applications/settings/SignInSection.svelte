<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import {
		FieldGrid,
		FormSection,
		Input,
		Note,
		Select,
		SwitchField,
		type SelectOption
	} from '#lib/components/ui/index.js';
	import { loginFlowChoicesOptions } from '#lib/query/index.js';
	import type { ApplicationForm } from '../form.svelte';

	type Props = { form: ApplicationForm };

	let { form }: Props = $props();

	/** The login flow an application follows when it names none. */
	const DEFAULT_FLOW = 'default';

	const flows = createQuery(() => loginFlowChoicesOptions());

	/** The default first, then every flow that is on, and the one already chosen even if off. */
	const flowOptions = $derived.by<SelectOption<string>[]>(() => {
		const all = flows.data?.flows ?? [];

		const options: SelectOption<string>[] = [
			{
				value: DEFAULT_FLOW,
				label: 'The default flow',
				description: all.find((flow) => flow.is_default)?.name ?? 'Whichever flow is the default'
			}
		];

		for (const flow of all) {
			if (flow.is_default) continue;
			if (!flow.is_enabled && flow.id !== form.input.login_flow_id) continue;

			options.push({
				value: flow.id,
				label: flow.name,
				description: flow.is_enabled ? flow.description : `${flow.description} · turned off`
			});
		}

		return options;
	});
</script>

<FormSection
	title="Sign-in page"
	description="What users see when this application sends them to sign in."
>
	<FieldGrid>
		<Input
			label="Terms of service URL"
			bind:value={form.input.terms_url}
			type="url"
			placeholder="https://shop.example.com/terms"
		/>
		<Input
			label="Privacy policy URL"
			bind:value={form.input.privacy_url}
			type="url"
			placeholder="https://shop.example.com/privacy"
		/>
	</FieldGrid>
	<Note boxed>
		Linked at the bottom of every sign-in page. When either is set, new users have to accept them to
		create an account.
	</Note>

	<SwitchField
		label="Allow registration"
		description="Offer “Create an account” on the sign-in page. New users get the default roles."
		bind:checked={form.input.allow_registration}
	/>

	{#if flowOptions.length > 1}
		<Select
			label="Login flow"
			value={form.input.login_flow_id || DEFAULT_FLOW}
			onChange={(chosen) => (form.input.login_flow_id = chosen === DEFAULT_FLOW ? '' : chosen)}
			options={flowOptions}
			hint="What this application's users are taken through when they sign in. Manage the flows themselves on the Login flows page."
		/>
	{/if}
</FormSection>
