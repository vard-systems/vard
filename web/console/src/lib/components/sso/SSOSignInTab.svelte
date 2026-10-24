<script lang="ts">
	import { FormSection, SwitchField } from '#lib/components/ui/index.js';
	import DomainsField from './DomainsField.svelte';
	import type { SSOForm } from './form.svelte';

	type Props = {
		form: SSOForm;
		readOnly: boolean;
	};

	let { form, readOnly }: Props = $props();
</script>

<FormSection
	title="Domains"
	description="The addresses at these domains are sent to this provider."
>
	<DomainsField
		bind:domains={() => form.domains, (domains) => (form.domains = domains)}
		{readOnly}
	/>
</FormSection>

<FormSection title="Sign-in" description="How people at those domains reach it.">
	<SwitchField
		label="Require SSO for these domains"
		description={form.domains.length > 0
			? 'Their password sign-in, registration and password reset are refused; the sign-in page sends them to the provider instead.'
			: 'Add a domain first: SSO is required of the addresses at its domains.'}
		bind:checked={() => form.enforced, (value) => (form.enforceDomains = value)}
		disabled={readOnly || form.domains.length === 0}
	/>
	<SwitchField
		label="Show a button on the sign-in page"
		description={`“Continue with ${form.name || '…'}”. Off, people reach it by typing their work address.`}
		bind:checked={form.showOnLogin}
		disabled={readOnly}
	/>
</FormSection>
