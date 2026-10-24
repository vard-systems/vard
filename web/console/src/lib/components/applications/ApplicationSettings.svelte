<script module lang="ts">
	import type { ApplicationType } from '#lib/api/index.js';
	import type { SelectOption } from '#lib/components/ui/index.js';
	import { types } from './applications';

	/** The login flow an application follows when it names none. */
	const DEFAULT_FLOW = 'default';

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
	import { resolve } from '$app/paths';
	import { createQuery } from '@tanstack/svelte-query';
	import type { Application } from '#lib/api/index.js';
	import {
		Button,
		FieldGrid,
		FormSection,
		Input,
		Select,
		SwitchField,
		Textarea
	} from '#lib/components/ui/index.js';
	import Choice from '#lib/components/roles/Choice.svelte';
	import { loginFlowChoicesOptions } from '#lib/query/index.js';
	import { formatDateTime } from '#lib/utils/format.js';
	import { authMethods, grants, scopes } from './applications';
	import type { ApplicationForm } from './form.svelte';

	type Props = {
		form: ApplicationForm;
		/** The saved application; null while a new one is being made. */
		current: Application | null;
		editable: boolean;
		saving: boolean;
		onrotate: () => void;
	};

	let { form, current, editable, saving, onrotate }: Props = $props();

	const flows = createQuery(() => loginFlowChoicesOptions());

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

	const rolesHref = $derived(
		current ? `${resolve('/(panel)/roles')}?tab=application&application=${current.id}` : ''
	);

	const offeredGrants = $derived(
		grants.filter((grant) => grant.value !== 'client_credentials' || !form.kind.public)
	);

	const maskedSecret = $derived(
		current?.has_secret ? `••••••••••••••••${current.secret_hint}` : 'No secret yet'
	);
</script>

<fieldset disabled={!editable}>
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

	<FormSection
		title="Credentials"
		description={form.kind.public
			? 'A public client cannot keep a secret, so it has none and must use PKCE.'
			: 'What the application proves who it is with at the token endpoint.'}
	>
		{#if current}
			<Input label="Client ID" value={current.client_id} readOnly copyable />
		{:else}
			<p class="note">
				The client ID{form.kind.public ? '' : ' and secret'} are made when you create the application.
			</p>
		{/if}

		{#if !form.kind.public}
			<Select
				label="Client authentication"
				bind:value={form.input.token_auth_method}
				options={authMethods}
			/>

			{#if current}
				<div class="secret">
					<Input
						label="Client secret"
						value={maskedSecret}
						readOnly
						hint={current.secret_created_at
							? `Created ${formatDateTime(current.secret_created_at)}. Only its last characters are kept.`
							: undefined}
					/>

					{#if editable}
						<div class="rotate">
							<Button size="sm" variant="subtle" onclick={onrotate} disabled={saving}>
								<i class="ri-refresh-line" aria-hidden="true"></i>
								Rotate secret
							</Button>
						</div>
					{/if}
				</div>
			{/if}
		{/if}
	</FormSection>

	{#if !form.machine}
		<FormSection
			title="Redirects"
			description="Where users may be sent back to. Each URI is matched exactly, with no wildcards."
		>
			<Textarea
				label="Redirect URIs"
				bind:value={form.redirects}
				rows={3}
				placeholder="https://shop.example.com/auth/callback"
				hint={form.input.type === 'native'
					? 'One per line. https, a loopback address, or a private-use scheme such as com.example.app:/callback.'
					: 'One per line. https, or http on localhost.'}
			/>

			<Textarea
				label="Post-logout redirect URIs"
				bind:value={form.logoutRedirects}
				rows={2}
				placeholder="https://shop.example.com/"
				hint="One per line. Where the app may send users after they sign out."
			/>
		</FormSection>

		<FormSection
			title="Grants and scopes"
			description="How the application signs users in, and what it may ask for."
		>
			<div class="options">
				<span class="options-label">Grant types</span>
				<div class="grid">
					{#each offeredGrants as grant (grant.value)}
						<Choice
							name={grant.value}
							description={grant.description}
							checked={form.input.grant_types.includes(grant.value)}
							onChange={(on) => form.setGrant(grant.value, on)}
						/>
					{/each}
				</div>
			</div>

			<div class="options">
				<span class="options-label">Scopes</span>
				<div class="grid">
					{#each scopes as scope (scope.value)}
						<Choice
							name={scope.value}
							description={scope.description}
							checked={form.input.scopes.includes(scope.value)}
							onChange={(on) => form.setScope(scope.value, on)}
						/>
					{/each}
				</div>
			</div>

			<SwitchField
				label="Require PKCE"
				description={form.kind.public
					? 'Always on for a public client.'
					: 'Recommended for every client: it stops a stolen code from being used.'}
				bind:checked={form.input.require_pkce}
				disabled={form.kind.public}
			/>
		</FormSection>
	{:else}
		<FormSection
			title="Grant"
			description="A machine-to-machine application uses client_credentials only: it has no user to sign in, so nothing to redirect to."
		>
			<Choice
				name="client_credentials"
				description="Act as the application itself, with no user"
				checked={true}
				onChange={() => {}}
				disabled
			/>
		</FormSection>
	{/if}

	<FormSection
		title="Token lifetimes"
		description="How long each token the application is given stays valid."
	>
		<div class="triple">
			<Input
				label="Access token"
				bind:value={form.accessMinutes}
				type="number"
				min="1"
				suffix="minutes"
			/>
			{#if !form.machine}
				<Input
					label="ID token"
					bind:value={form.idMinutes}
					type="number"
					min="1"
					suffix="minutes"
				/>
				<Input
					label="Refresh token"
					bind:value={form.refreshDays}
					type="number"
					min="1"
					suffix="days"
				/>
			{/if}
		</div>
	</FormSection>

	{#if !form.machine}
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
			<p class="note">
				Linked at the bottom of every sign-in page. When either is set, new users have to accept
				them to create an account.
			</p>

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

		<FormSection title="Roles" description="How the application's roles reach its tokens.">
			{#snippet action()}
				{#if current}
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a class="link" href={rolesHref}>
						Manage {current.role_count}
						{current.role_count === 1 ? 'role' : 'roles'}
						<i class="ri-arrow-right-line" style:font-size="14px" aria-hidden="true"></i>
					</a>
				{/if}
			{/snippet}

			<div class="switches">
				<SwitchField
					label="Include roles in tokens"
					description="Put the user's roles in this application into its tokens and userinfo."
					bind:checked={form.input.assert_roles}
				/>
				<SwitchField
					label="Require a role to sign in"
					description="Only let users who hold at least one of its roles sign in to it."
					bind:checked={form.input.require_role_assignment}
				/>
			</div>
		</FormSection>
	{/if}
</fieldset>

<style>
	fieldset {
		min-width: 0;
		margin: 0;
		padding: 0;
		border: none;
	}

	.note {
		margin: 0;
		padding: var(--space-3);
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.triple {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
		gap: var(--space-3);
	}

	.secret {
		display: grid;
		grid-template-columns: 1fr auto;
		align-items: start;
		gap: var(--space-3);
	}

	.secret :global([data-part='input']) {
		font-family: var(--font-mono);
		letter-spacing: 0.02em;
	}

	.rotate {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: var(--space-2);
		padding-top: 14px;
	}

	.options {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.options-label {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-weight: 700;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr));
		gap: 2px var(--space-3);
	}

	.switches {
		display: flex;
		flex-direction: column;
	}

	.link {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		color: var(--color-text);
		font-size: var(--text-sm);
		font-weight: 600;
		text-decoration: none;
		white-space: nowrap;
	}

	.link:hover {
		text-decoration: underline;
	}

	@media (max-width: 36rem) {
		.secret {
			grid-template-columns: 1fr;
		}

		.rotate {
			align-items: flex-start;
			padding-top: 0;
		}
	}
</style>
