<script lang="ts">
	import {
		FormSection,
		Input,
		Note,
		Select,
		SwitchField,
		Textarea
	} from '#lib/components/ui/index.js';
	import DomainProofs from './DomainProofs.svelte';
	import DomainsField from './DomainsField.svelte';
	import type { SSOForm } from './form.svelte';

	type Props = {
		form: SSOForm;
		readOnly: boolean;
		/** The saved connection's id; its domains are proved once it exists. */
		connectionId?: string;
	};

	let { form, readOnly, connectionId }: Props = $props();

	const secondFactors = [
		{ value: 'local', label: 'This server' },
		{ value: 'provider', label: 'The provider' }
	] as const;
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

{#if connectionId && form.domains.length > 0}
	<FormSection
		title="Domain proofs"
		description="Proof that the domains are the organisation's, before the provider may take over accounts there."
	>
		<DomainProofs {connectionId} canWrite={!readOnly} />
	</FormSection>
{/if}

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

<FormSection
	title="Second factor"
	description="Asked only by a login flow that requires two-step verification."
>
	<Select
		label="Second factor"
		bind:value={form.secondFactor}
		options={secondFactors}
		hint={form.secondFactor === 'local'
			? 'People are asked for their account’s own second factor here, after the provider.'
			: 'The provider’s own second factor counts when its answer shows one.'}
		{readOnly}
	/>
	{#if form.secondFactor === 'provider'}
		{#if form.protocol === 'oidc'}
			<Input
				label="Accepted amr values"
				bind:value={form.acceptedAmr}
				hint="Space separated. The id_token’s amr has to hold one of them; mfa is the standard one."
				placeholder="mfa"
				{readOnly}
			/>
		{:else}
			<Textarea
				label="Authentication contexts"
				bind:value={form.mfaContexts}
				rows={3}
				hint="One per line: the AuthnContextClassRef values that mean the provider asked for a second factor. SAML names no standard one."
			/>
		{/if}
		<Note>
			When the provider’s answer shows none of them, the account’s own second factor is asked for
			instead, and the activity log says so.
		</Note>
	{/if}
</FormSection>
