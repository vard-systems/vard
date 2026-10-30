<script lang="ts">
	import { enhance } from '$app/forms';
	import { invalidate } from '$app/navigation';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import type { Organization } from '#lib/api/index.js';
	import {
		Alert,
		FieldGrid,
		FormSection,
		Input,
		SaveBar,
		Select,
		Tag,
		notify
	} from '#lib/components/ui/index.js';
	import { ADMIN_DEPENDENCY } from '#lib/constants.js';
	import { submit } from '#lib/forms.js';
	import { keys, organizationOptions } from '#lib/query/index.js';
	import LogoField from './LogoField.svelte';
	import OrganizationSummary from './OrganizationSummary.svelte';
	import { changed, offsetOf, settingsOf, tidy, timezoneOptions } from './settings';

	type Props = {
		initial: Organization;
		editable: boolean;
	};

	let { initial, editable }: Props = $props();

	const queryClient = useQueryClient();

	const settings = createQuery(() => organizationOptions({ organization: initial }));

	const organization = $derived(settings.data.organization);

	// Seeded once, when the panel is first drawn: a refetch in the background
	// leaves what is being typed alone, and a save fills it in again.
	// svelte-ignore state_referenced_locally
	let form = $state(settingsOf(organization));

	let saving = $state(false);

	function discard() {
		form = settingsOf(organization);
	}

	const input = $derived(tidy(form));
	const timezones = $derived(timezoneOptions(organization.timezone));
	const dirty = $derived(changed(input, organization));
	const complete = $derived(input.name !== '' && input.slug !== '');
	const offset = $derived(offsetOf(input.timezone));

	const save = submit<{ organization: Organization }>({
		input: () => input,
		when: () => editable && !saving && dirty && complete,
		pending: (busy) => (saving = busy),
		done: async (result) => {
			queryClient.setQueryData(keys.organization.settings, result);
			form = settingsOf(result.organization);
			notify.success('Settings saved', 'The sign-in pages and the discovery document follow.');
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.admin.overview }),
				invalidate(ADMIN_DEPENDENCY)
			]);
		},
		failed: (err) => notify.error(err, 'Could not save these settings')
	});

	// The logo is saved the moment it is chosen, apart from the form: the
	// settings view takes the new version, and the header reloads its own.
	async function logoSaved(saved: Organization) {
		queryClient.setQueryData(keys.organization.settings, { organization: saved });
		await invalidate(ADMIN_DEPENDENCY);
	}
</script>

<form method="POST" action="/actions/organization?/update" use:enhance={save}>
	<OrganizationSummary {input} saved={organization} />

	{#if !editable}
		<Alert tone="info">Your roles let you see these settings but not change them.</Alert>
	{/if}

	<fieldset disabled={!editable}>
		<FormSection
			title="Identity"
			description="What the organization is called, and the logo this panel's header and the sign-in pages show."
			icon="ri-building-line"
		>
			<FieldGrid spacing="comfortable">
				<Input
					label="Name"
					bind:value={form.name}
					required
					maxlength={100}
					hint="Shown on the sign-in pages."
				/>

				<Input
					label="Identifier"
					bind:value={form.slug}
					required
					maxlength={64}
					hint="Lower case letters, numbers and dashes."
				/>
			</FieldGrid>

			<LogoField version={organization.logo_version} {editable} onsaved={logoSaved} />
		</FormSection>

		<FormSection
			title="Region"
			description="The zone dates are written in on a user's account pages, such as when the account was made."
			icon="ri-global-line"
		>
			{#snippet meta()}
				{#if offset}<Tag small>{offset}</Tag>{/if}
			{/snippet}

			<Select
				label="Timezone"
				icon="ri-time-line"
				bind:value={form.timezone}
				options={timezones}
				required
				disabled={!editable}
			/>
		</FormSection>

		<FormSection
			title="Support"
			description="Where someone who cannot get in can ask for help."
			icon="ri-customer-service-2-line"
		>
			{#snippet meta()}
				<Tag small>Shown when signing in</Tag>
			{/snippet}

			<FieldGrid spacing="comfortable">
				<Input
					label="Support email"
					icon="ri-mail-line"
					bind:value={form.support_email}
					type="email"
					maxlength={255}
					placeholder="support@example.com"
				/>

				<Input
					label="Support number"
					icon="ri-phone-line"
					bind:value={form.support_phone}
					maxlength={32}
					placeholder="+996 555 123456"
					hint="In the form you want it dialled."
				/>
			</FieldGrid>
		</FormSection>

		<FormSection
			title="Agreements"
			description="What users agree to. An application with links of its own shows those instead. A login flow with the Agreements step asks users to accept these, and again when they change."
			icon="ri-file-text-line"
		>
			{#snippet meta()}
				<Tag small>Sign-in pages</Tag>
				<Tag small>Discovery document</Tag>
			{/snippet}

			<Input
				label="Terms of service"
				icon="ri-file-text-line"
				bind:value={form.terms_url}
				maxlength={512}
				placeholder="https://example.com/terms"
				hint="Published as op_tos_uri, and linked under the sign-in card."
			/>

			<Input
				label="Privacy policy"
				icon="ri-shield-check-line"
				bind:value={form.privacy_url}
				maxlength={512}
				placeholder="https://example.com/privacy"
				hint="Published as op_policy_uri, and agreed to on registration."
			/>
		</FormSection>
	</fieldset>

	{#if editable && dirty}
		<SaveBar {saving} ready={complete} ondiscard={discard} />
	{/if}
</form>

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-5);
	}

	fieldset {
		min-width: 0;
		margin: 0;
		padding: 0;
		border: none;
	}
</style>
