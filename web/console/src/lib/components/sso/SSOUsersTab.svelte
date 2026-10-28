<script module lang="ts">
	const matchings = [
		{ value: 'link', label: 'Link it: the provider owns the domain' },
		{ value: 'deny', label: 'Refuse: it signs in the way it was made' }
	];

	/** The names each protocol usually carries them under, shown when none is given. */
	const usual = {
		oidc: { email: 'email', groups: 'groups', firstName: 'given_name', lastName: 'family_name' },
		saml: { email: 'mail', groups: 'memberOf', firstName: 'givenName', lastName: 'surname' }
	} as const;
</script>

<script lang="ts">
	import type { Role } from '#lib/api/index.js';
	import { FieldGrid, FormSection, Input, Select, SwitchField } from '#lib/components/ui/index.js';
	import type { SSOForm } from './form.svelte';
	import RoleMappings from './RoleMappings.svelte';

	type Props = {
		form: SSOForm;
		roles: Role[];
		canReadRoles: boolean;
		readOnly: boolean;
	};

	let { form, roles, canReadRoles, readOnly }: Props = $props();

	const placeholders = $derived(usual[form.protocol]);
</script>

<FormSection
	title="Provisioning"
	description="What happens to an account when somebody signs in through it."
>
	<Select
		label="An address that already has an account"
		bind:value={
			() => (form.domains.length === 0 ? 'deny' : form.matching),
			(value) => (form.matching = value as typeof form.matching)
		}
		options={matchings}
		hint={form.domains.length === 0
			? 'A connection without domains never links to existing accounts: add its domains first.'
			: 'It links only at a domain proved on Domains & sign-in.'}
		readOnly={readOnly || form.domains.length === 0}
	/>
	<SwitchField
		label="Create accounts on first sign-in"
		description="Just-in-time provisioning. Off, only people who already have an account here can sign in."
		bind:checked={form.createUsers}
		disabled={readOnly}
	/>
	<SwitchField
		label="Update names on every sign-in"
		description="The provider is where they are kept, so a change there reaches here."
		bind:checked={form.syncProfile}
		disabled={readOnly}
	/>
</FormSection>

<FormSection
	title="Where to read"
	description="The claim or attribute names. Empty reads the usual ones."
>
	<FieldGrid>
		<Input
			label="Email"
			bind:value={form.emailAttribute}
			placeholder={placeholders.email}
			{readOnly}
		/>
		<Input
			label="Groups"
			bind:value={form.groupsAttribute}
			placeholder={placeholders.groups}
			{readOnly}
		/>
		<Input
			label="First name"
			bind:value={form.firstNameAttribute}
			placeholder={placeholders.firstName}
			{readOnly}
		/>
		<Input
			label="Last name"
			bind:value={form.lastNameAttribute}
			placeholder={placeholders.lastName}
			{readOnly}
		/>
	</FieldGrid>
</FormSection>

<FormSection
	title="Group to role mapping"
	description="Everybody the provider puts in a group gets the role, at every sign-in."
>
	<RoleMappings
		bind:mappings={() => form.mappings, (mappings) => (form.mappings = mappings)}
		{roles}
		canRead={canReadRoles}
		{readOnly}
	/>
	<SwitchField
		label="Take mapped roles away too"
		description="Someone no longer in a group loses its role at their next sign-in. Off, mapped roles are only ever added."
		bind:checked={form.syncRoles}
		disabled={readOnly}
	/>
</FormSection>
