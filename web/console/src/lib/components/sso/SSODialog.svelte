<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import {
		RiCheckLine,
		RiDownload2Line,
		RiFlaskLine,
		RiGroupLine,
		RiLinksLine,
		RiRefreshLine,
		RiServerLine,
		RiSettings3Line
	} from 'svelte-remixicon';
	import {
		messageOf,
		ssoApi,
		type Role,
		type SSOConnection,
		type SSOConnectionInput,
		type SSOProtocol,
		type SSORoleMapping,
		type SSOTestResult
	} from '$lib/api';
	import {
		Alert,
		Button,
		ConfirmDialog,
		DangerZone,
		FullscreenDialog,
		FieldGrid,
		FormSection,
		Icon,
		Input,
		PasswordInput,
		Select,
		SwitchField,
		Tabs,
		Textarea,
		notify
	} from '$lib/components/ui';
	import { keys } from '$lib/query';
	import DomainsField from './DomainsField.svelte';
	import RoleMappings from './RoleMappings.svelte';

	type Props = {
		open: boolean;
		connection?: SSOConnection | null;
		roles: Role[];
		canReadRoles: boolean;
		canWrite: boolean;
	};

	let {
		open = $bindable(false),
		connection = null,
		roles,
		canReadRoles,
		canWrite
	}: Props = $props();

	const queryClient = useQueryClient();

	let current = $state<SSOConnection | null>(null);

	let tab = $state('connection');
	let protocol = $state<SSOProtocol>('oidc');
	let name = $state('');
	let slug = $state('');
	let enabled = $state(false);
	let issuer = $state('');
	let clientId = $state('');
	let clientSecret = $state('');
	let scopes = $state('');
	let metadataUrl = $state('');
	let metadata = $state('');
	let nameIdFormat = $state<SSOConnection['name_id_format']>('email');
	let signRequests = $state(false);
	let domains = $state<string[]>([]);
	let enforceDomains = $state(false);
	let showOnLogin = $state(false);
	let matching = $state<SSOConnection['matching']>('link');
	let createUsers = $state(true);
	let syncProfile = $state(true);
	let emailAttribute = $state('');
	let firstNameAttribute = $state('');
	let lastNameAttribute = $state('');
	let groupsAttribute = $state('');
	let mappings = $state<SSORoleMapping[]>([]);
	let syncRoles = $state(false);

	let saving = $state(false);
	let confirmingDelete = $state(false);
	let tested = $state<SSOTestResult | null>(null);
	let testError = $state('');
	let testing = $state(false);

	$effect(() => {
		if (!open) return;

		current = connection;
		fill(connection);
		tab = 'connection';
		confirmingDelete = false;
		tested = null;
		testError = '';
	});

	function fill(from: SSOConnection | null) {
		protocol = from?.protocol ?? 'oidc';
		name = from?.name ?? '';
		slug = from?.slug ?? '';
		enabled = from?.is_enabled ?? false;
		issuer = from?.issuer ?? '';
		clientId = from?.client_id ?? '';
		clientSecret = '';
		scopes = (from?.scopes ?? []).join(' ');
		metadataUrl = from?.metadata_url ?? '';
		metadata = from?.metadata_url ? '' : (from?.metadata ?? '');
		nameIdFormat = from?.name_id_format ?? 'email';
		signRequests = from?.sign_requests ?? false;
		domains = [...(from?.domains ?? [])];
		enforceDomains = from?.enforce_domains ?? false;
		showOnLogin = from?.show_on_login ?? true;
		matching = from?.matching ?? 'link';
		createUsers = from?.create_users ?? true;
		syncProfile = from?.sync_profile ?? true;
		emailAttribute = from?.email_attribute ?? '';
		firstNameAttribute = from?.first_name_attribute ?? '';
		lastNameAttribute = from?.last_name_attribute ?? '';
		groupsAttribute = from?.groups_attribute ?? '';
		mappings = (from?.role_mappings ?? []).map((mapping) => ({ ...mapping }));
		syncRoles = from?.sync_roles ?? false;
	}

	const editing = $derived(current !== null);
	const readOnly = $derived(!canWrite);

	const enforced = $derived(domains.length > 0 && enforceDomains);

	const missing = $derived.by((): { text: string; tab: string } | null => {
		if (name.trim() === '') return { text: 'Name the connection to create it.', tab: 'connection' };
		if (protocol === 'oidc' && (issuer.trim() === '' || clientId.trim() === ''))
			return { text: 'Give the issuer and the client ID to create it.', tab: 'connection' };
		if (protocol === 'saml' && metadataUrl.trim() === '' && metadata.trim() === '')
			return {
				text: 'Give the metadata URL or paste the metadata to create it.',
				tab: 'connection'
			};
		if (domains.length === 0 && !showOnLogin)
			return {
				text: 'Add a domain or turn on the sign-in button, on Domains & sign-in, to create it.',
				tab: 'signin'
			};
		return null;
	});

	function input(): SSOConnectionInput {
		const body: SSOConnectionInput = {
			name: name.trim(),
			is_enabled: enabled,
			domains,
			enforce_domains: enforced,
			show_on_login: showOnLogin,
			matching,
			create_users: createUsers,
			sync_profile: syncProfile,
			email_attribute: emailAttribute.trim(),
			first_name_attribute: firstNameAttribute.trim(),
			last_name_attribute: lastNameAttribute.trim(),
			groups_attribute: groupsAttribute.trim(),
			role_mappings: mappings.filter((mapping) => mapping.group.trim() !== ''),
			sync_roles: syncRoles
		};

		if (!editing) {
			body.protocol = protocol;
			if (slug.trim()) body.slug = slug.trim();
		}

		if (protocol === 'oidc') {
			body.issuer = issuer.trim();
			body.client_id = clientId.trim();
			body.scopes = scopes.split(/\s+/).filter(Boolean);
			if (clientSecret.trim()) body.client_secret = clientSecret.trim();
		} else {
			body.metadata_url = metadataUrl.trim();
			if (!metadataUrl.trim()) body.metadata = metadata.trim();
			body.name_id_format = nameIdFormat;
			body.sign_requests = signRequests;
		}

		return body;
	}

	const save = createMutation(() => ({
		mutationFn: () => (current ? ssoApi.update(current.id, input()) : ssoApi.create(input())),
		onSuccess: async ({ connection: saved }) => {
			const created = !editing;
			notify.success(created ? 'Connection created' : 'Connection saved');
			await queryClient.invalidateQueries({ queryKey: keys.sso.all });

			if (created) {
				current = saved;
				fill(saved);
				tab = 'provider';
				return;
			}

			open = false;
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this connection');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	const remove = createMutation(() => ({
		mutationFn: () => ssoApi.remove(current?.id ?? ''),
		onSuccess: async () => {
			notify.success('Connection removed');
			open = false;
			await queryClient.invalidateQueries({ queryKey: keys.sso.all });
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not remove this connection');
		}
	}));

	const refresh = createMutation(() => ({
		mutationFn: () => ssoApi.refreshMetadata(current?.id ?? ''),
		onSuccess: async ({ connection: saved }) => {
			notify.success('Metadata read again');
			current = saved;
			await queryClient.invalidateQueries({ queryKey: keys.sso.all });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not read the metadata again');
		}
	}));

	async function test() {
		testing = true;
		testError = '';
		tested = null;

		try {
			tested = await ssoApi.test({
				protocol,
				issuer: issuer.trim(),
				scopes: scopes.split(/\s+/).filter(Boolean),
				metadata_url: metadataUrl.trim(),
				metadata: metadataUrl.trim() ? undefined : metadata.trim()
			});
		} catch (err) {
			testError = messageOf(err);
		} finally {
			testing = false;
		}
	}

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (missing || saving || readOnly) return;

		saving = true;
		save.mutate();
	}

	const tabs = $derived([
		{ value: 'connection', label: 'Connection', icon: RiSettings3Line },
		{ value: 'provider', label: 'Service provider', icon: RiServerLine },
		{ value: 'signin', label: 'Domains & sign-in', icon: RiLinksLine },
		{ value: 'users', label: 'Users & roles', icon: RiGroupLine }
	]);

	const protocols = $derived([
		{
			value: 'oidc',
			label: 'OpenID Connect',
			description: 'Okta, Entra ID, Google Workspace, Keycloak, Auth0 and most modern providers'
		},
		{
			value: 'saml',
			label: 'SAML 2.0',
			description: 'ADFS, Entra ID, Okta, OneLogin, Shibboleth and most enterprise providers'
		}
	]);

	const nameIdFormats = $derived([
		{ value: 'email', label: 'Email address' },
		{ value: 'persistent', label: 'Persistent' },
		{ value: 'unspecified', label: 'Unspecified' }
	]);

	const matchings = $derived([
		{ value: 'link', label: 'Link it: the provider owns the domain' },
		{ value: 'deny', label: 'Refuse: it signs in the way it was made' }
	]);

	const expiring = $derived.by(() => {
		const expires = current?.identity_provider?.certificate_expires;
		if (!expires) return null;

		const when = new Date(expires);
		return when.getTime() - Date.now() < 30 * 24 * 3600 * 1000 ? when : null;
	});

	function download(filename: string, text: string, type: string) {
		const url = URL.createObjectURL(new Blob([text], { type }));
		const link = document.createElement('a');
		link.href = url;
		link.download = filename;
		link.click();
		URL.revokeObjectURL(url);
	}
</script>

{#snippet testButton()}
	<Button variant="subtle" size="sm" onclick={test} loading={testing} disabled={testing}>
		<Icon icon={RiFlaskLine} />
		Test connection
	</Button>
{/snippet}

{#snippet testResult()}
	{#if testError}
		<Alert>{testError}</Alert>
	{:else if tested?.authorization_endpoint}
		<Alert tone="success">
			{`Discovery worked: people are sent to ${tested.authorization_endpoint} to sign in.`}
		</Alert>
		{#if tested.unsupported_scopes?.length}
			<Alert tone="warning">
				{`The provider does not list ${tested.unsupported_scopes.join(' ')} among its scopes, and some providers — Keycloak among them — refuse the whole sign-in over a scope they do not know. Remove it, or add it at the provider first.`}
			</Alert>
		{/if}
	{:else if tested?.identity_provider}
		<Alert tone="success">
			{`The metadata is ${tested.identity_provider.entity_id}'s: people are sent to ${tested.identity_provider.sso_url} to sign in.`}
		</Alert>
	{/if}
{/snippet}

{#snippet status()}
	{#if missing}
		<button type="button" class="missing" onclick={() => (tab = missing.tab)}>
			{missing.text}
		</button>
	{/if}
{/snippet}

<FullscreenDialog
	bind:open
	status={canWrite && missing ? status : undefined}
	title={current ? current.name : 'New connection'}
	description={current
		? 'Where it signs people in, for which domains, and what they become here.'
		: "An organisation's identity provider. It starts off, so it can be set up and tested before anybody signs in through it."}
	meta={current?.slug}
	onsubmit={submit}
>
	<Tabs {tabs} bind:value={tab} label="SSO integrations">
		{#snippet panel(value)}
			<div class="panel">
				{#if value === 'connection'}
					<FormSection
						title="Connection"
						description="What the sign-in page calls it, and whether it is on."
					>
						{#if !editing}
							<Select label="Protocol" bind:value={protocol} options={protocols} />
						{/if}

						<FieldGrid>
							<Input
								label="Name"
								bind:value={name}
								hint={`What the sign-in page calls it: “Continue with ${name.trim() || 'Acme'}”.`}
								required
								{readOnly}
							/>
							<Input
								label="Identifier"
								bind:value={slug}
								hint="In the addresses the provider is given, so it cannot change. Made from the name unless you give one."
								placeholder="acme"
								readOnly={editing || readOnly}
							/>
						</FieldGrid>

						<SwitchField
							label="On"
							description="Off, nobody signs in through it — set it up and test it first."
							bind:checked={enabled}
							disabled={readOnly}
						/>
					</FormSection>

					{#if protocol === 'oidc'}
						<FormSection
							title="OpenID Connect"
							description="Register this server at the provider as a web application, then copy its issuer and credentials here."
							action={canWrite ? testButton : undefined}
						>
							<Input
								label="Issuer"
								bind:value={issuer}
								hint="Its discovery document is read from the issuer's /.well-known/openid-configuration."
								placeholder="https://acme.okta.com"
								required
								{readOnly}
							/>
							<FieldGrid>
								<Input label="Client ID" bind:value={clientId} required {readOnly} />
								<div class="secret">
									<PasswordInput
										label="Client secret"
										bind:value={clientSecret}
										autocomplete="new-password"
										disabled={readOnly}
									/>
									{#if current?.has_client_secret}
										<p class="note small">A secret is stored. Type a new one to replace it.</p>
									{/if}
								</div>
							</FieldGrid>
							<Input
								label="Scopes"
								bind:value={scopes}
								hint="Space separated. openid is always asked for; email and profile unless you name others."
								placeholder="email profile groups"
								{readOnly}
							/>
							{@render testResult()}
						</FormSection>
					{:else}
						<FormSection
							title="SAML 2.0"
							description="Give the provider this server's metadata, then its own here: from an address, or the file it gave you."
							action={canWrite ? testButton : undefined}
						>
							<Input
								label="Metadata URL"
								bind:value={metadataUrl}
								hint="Read when you save, and again with Refresh metadata."
								placeholder="https://login.microsoftonline.com/…/federationmetadata.xml"
								{readOnly}
							/>
							{#if !metadataUrl.trim()}
								<Textarea
									label="Metadata XML"
									bind:value={metadata}
									rows={5}
									hint="Or paste the file the provider gave you."
								/>
							{/if}
							<FieldGrid>
								<Select
									label="Name ID format"
									bind:value={nameIdFormat}
									options={nameIdFormats}
									{readOnly}
								/>
							</FieldGrid>
							<SwitchField
								label="Sign authentication requests"
								description="For providers that require it. The certificate is in this server's metadata."
								bind:checked={signRequests}
								disabled={readOnly}
							/>
							{@render testResult()}
						</FormSection>
					{/if}

					{#if canWrite && current}
						<DangerZone
							title="Remove this connection"
							description="The people who sign in through it keep their accounts, and sign in however else they can."
							onclick={() => (confirmingDelete = true)}
						/>
					{/if}
				{:else if value === 'provider'}
					{#if !current}
						<p class="note">Create the connection to get the addresses its provider needs.</p>
					{:else if current.protocol === 'oidc'}
						<FormSection
							title="What to give the provider"
							description="Register this redirect URI with the application at the provider."
						>
							<Input
								label="Redirect URI"
								value={current.service_provider.callback_url ?? ''}
								readOnly
								copyable
							/>
						</FormSection>
					{:else}
						<FormSection
							title="What to give the provider"
							description="Give the provider the metadata URL — or, if it asks for them one by one, these."
						>
							<Input
								label="ACS URL (reply URL)"
								value={current.service_provider.acs_url ?? ''}
								readOnly
								copyable
							/>
							<Input
								label="Entity ID (audience)"
								value={current.service_provider.entity_id ?? ''}
								readOnly
								copyable
							/>
							<Input
								label="Service provider metadata"
								value={current.service_provider.metadata_url ?? ''}
								readOnly
								copyable
							/>

							<div class="files">
								<Button
									variant="subtle"
									size="sm"
									onclick={() =>
										download(
											`${current!.slug}-certificate.pem`,
											current!.service_provider.certificate ?? '',
											'application/x-pem-file'
										)}
								>
									<Icon icon={RiDownload2Line} />
									Download signing certificate
								</Button>
							</div>
						</FormSection>

						{#if current.identity_provider}
							<FormSection
								title="The identity provider"
								description="What its metadata says about it."
							>
								<dl class="facts">
									<dt>Entity ID</dt>
									<dd><code>{current.identity_provider.entity_id}</code></dd>
									<dt>Sign-in URL</dt>
									<dd><code>{current.identity_provider.sso_url}</code></dd>
									{#if current.identity_provider.certificate_expires}
										<dt>Certificate expires</dt>
										<dd>
											{new Date(current.identity_provider.certificate_expires).toLocaleDateString()}
										</dd>
									{/if}
								</dl>

								{#if expiring}
									<Alert tone="warning"
										>{`Its certificate expires on ${expiring.toLocaleDateString()}. Refresh the metadata once the provider has rolled it over.`}</Alert
									>
								{/if}

								{#if canWrite && current.metadata_url}
									<div>
										<Button
											variant="subtle"
											size="sm"
											onclick={() => refresh.mutate()}
											loading={refresh.isPending}
										>
											<Icon icon={RiRefreshLine} />
											Refresh metadata
										</Button>
									</div>
								{/if}
							</FormSection>
						{/if}
					{/if}
				{:else if value === 'signin'}
					<FormSection
						title="Domains"
						description="The addresses at these domains are sent to this provider."
					>
						<DomainsField bind:domains {readOnly} />
					</FormSection>

					<FormSection title="Sign-in" description="How people at those domains reach it.">
						<SwitchField
							label="Require SSO for these domains"
							description={domains.length > 0
								? 'Their password sign-in, registration and password reset are refused; the sign-in page sends them to the provider instead.'
								: 'Add a domain first: SSO is required of the addresses at its domains.'}
							bind:checked={() => enforced, (value) => (enforceDomains = value)}
							disabled={readOnly || domains.length === 0}
						/>
						<SwitchField
							label="Show a button on the sign-in page"
							description={`“Continue with ${name || '…'}”. Off, people reach it by typing their work address.`}
							bind:checked={showOnLogin}
							disabled={readOnly}
						/>
					</FormSection>
				{:else}
					<FormSection
						title="Provisioning"
						description="What happens to an account when somebody signs in through it."
					>
						<Select
							label="An address that already has an account"
							bind:value={matching}
							options={matchings}
							{readOnly}
						/>
						<SwitchField
							label="Create accounts on first sign-in"
							description="Just-in-time provisioning. Off, only people who already have an account here can sign in."
							bind:checked={createUsers}
							disabled={readOnly}
						/>
						<SwitchField
							label="Update names on every sign-in"
							description="The provider is where they are kept, so a change there reaches here."
							bind:checked={syncProfile}
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
								bind:value={emailAttribute}
								placeholder={protocol === 'oidc' ? 'email' : 'mail'}
								{readOnly}
							/>
							<Input
								label="Groups"
								bind:value={groupsAttribute}
								placeholder={protocol === 'oidc' ? 'groups' : 'memberOf'}
								{readOnly}
							/>
							<Input
								label="First name"
								bind:value={firstNameAttribute}
								placeholder={protocol === 'oidc' ? 'given_name' : 'givenName'}
								{readOnly}
							/>
							<Input
								label="Last name"
								bind:value={lastNameAttribute}
								placeholder={protocol === 'oidc' ? 'family_name' : 'surname'}
								{readOnly}
							/>
						</FieldGrid>
					</FormSection>

					<FormSection
						title="Group to role mapping"
						description="Everybody the provider puts in a group gets the role, at every sign-in."
					>
						<RoleMappings bind:mappings {roles} canRead={canReadRoles} {readOnly} />
						<SwitchField
							label="Take mapped roles away too"
							description="Someone no longer in a group loses its role at their next sign-in. Off, mapped roles are only ever added."
							bind:checked={syncRoles}
							disabled={readOnly}
						/>
					</FormSection>
				{/if}
			</div>
		{/snippet}
	</Tabs>

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)}>Cancel</Button>
		{#if canWrite}
			<Button type="submit" loading={saving} disabled={missing !== null || saving}>
				<Icon icon={RiCheckLine} />
				{editing ? 'Save changes' : 'Create connection'}
			</Button>
		{/if}
	{/snippet}
</FullscreenDialog>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Remove ${current?.name ?? 'this connection'}?`}
	description={`${current?.users ?? 0} people sign in through it. They keep their accounts, and sign in however else they can.`}
	confirmLabel="Remove connection"
	busy={remove.isPending}
	onConfirm={() => remove.mutate()}
/>

<style>
	.panel {
		padding-top: var(--space-4);
	}

	.files {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: var(--space-2);
	}

	.note {
		margin: var(--space-4) 0;
		color: var(--color-text-hint);
	}

	.note.small {
		margin: var(--space-1) 0 0;
		font-size: var(--text-sm);
	}

	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: var(--space-1) var(--space-3);
		margin: 0;
		font-size: var(--text-sm);
	}

	.facts dt {
		color: var(--color-text-hint);
	}

	.facts dd {
		margin: 0;
		overflow-wrap: anywhere;
	}

	.facts code {
		font-family: var(--font-mono);
	}

	.missing {
		all: unset;
		cursor: pointer;
	}

	.missing:hover {
		color: var(--color-text);
		text-decoration: underline;
	}

	.missing:focus-visible {
		outline: 2px solid var(--color-info);
		outline-offset: 2px;
		border-radius: var(--radius-sm);
	}
</style>
