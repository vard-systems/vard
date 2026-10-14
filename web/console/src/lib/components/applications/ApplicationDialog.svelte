<script lang="ts">
	import { resolve } from '$app/paths';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import {
		RiArrowRightLine,
		RiCodeBoxLine,
		RiRefreshLine,
		RiSettings3Line,
		RiShieldCheckLine
	} from 'svelte-remixicon';
	import {
		applicationsApi,
		type Admin,
		type Application,
		type ApplicationInput,
		type ApplicationType,
		type ApplicationWithSecret,
		type GrantType,
		type Scope
	} from '#lib/api/index.js';
	import {
		Badge,
		Button,
		ConfirmDialog,
		FullscreenDialog,
		FieldGrid,
		FormSection,
		Icon,
		Input,
		Select,
		type SelectOption,
		SwitchField,
		Tabs,
		Textarea,
		notify
	} from '#lib/components/ui/index.js';
	import Choice from '#lib/components/roles/Choice.svelte';
	import { keys, loginFlowChoicesOptions } from '#lib/query/index.js';
	import { formatDateTime } from '#lib/utils/format.js';
	import { authMethods, blank, grants, lines, scopes, types } from './applications';
	import ApplicationApis from './ApplicationApis.svelte';
	import SecretPanel from './SecretPanel.svelte';
	import TokenPreviewPanel from './TokenPreviewPanel.svelte';

	type Props = {
		application: Application | null;
		editable: boolean;
		admin: Admin;
		open: boolean;
	};

	let { application, editable, admin, open = $bindable(false) }: Props = $props();

	let tab = $state('settings');

	const queryClient = useQueryClient();

	let current = $state<Application | null>(null);

	let secret = $state('');

	let form = $state<ApplicationInput>(blank('web'));
	let redirects = $state('');
	let logoutRedirects = $state('');

	let accessMinutes = $state('60');
	let idMinutes = $state('60');
	let refreshDays = $state('30');

	let saving = $state(false);
	let confirmingRotate = $state(false);

	const editing = $derived(current !== null);
	const kind = $derived(types[form.type]);
	const machine = $derived(form.type === 'm2m');

	const flows = createQuery(() => loginFlowChoicesOptions());

	const DEFAULT_FLOW = 'default';

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
			if (!flow.is_enabled && flow.id !== form.login_flow_id) continue;

			options.push({
				value: flow.id,
				label: flow.name,
				description: flow.is_enabled ? flow.description : `${flow.description} · turned off`
			});
		}

		return options;
	});

	const typeOptions: SelectOption<ApplicationType>[] = (
		Object.entries(types) as [ApplicationType, (typeof types)[ApplicationType]][]
	).map(([value, meta]) => ({
		value,
		label: meta.label,
		description: meta.description,
		icon: meta.icon
	}));

	$effect(() => {
		if (!open) {
			tab = 'settings';
			return;
		}

		current = application;
		secret = '';
		confirmingRotate = false;
		fill(application ? asInput(application) : blank('web'));
	});

	function asInput(app: Application): ApplicationInput {
		return { ...app, login_flow_id: app.login_flow_id ?? '' };
	}

	function fill(input: ApplicationInput) {
		form = {
			...input,
			grant_types: [...input.grant_types],
			scopes: [...input.scopes],
			redirect_uris: [...input.redirect_uris],
			post_logout_redirect_uris: [...input.post_logout_redirect_uris]
		};
		redirects = input.redirect_uris.join('\n');
		logoutRedirects = input.post_logout_redirect_uris.join('\n');
		accessMinutes = String(Math.round(input.access_token_lifetime / 60));
		idMinutes = String(Math.round(input.id_token_lifetime / 60));
		refreshDays = String(Math.round(input.refresh_token_lifetime / 86400));
	}

	function changeType(type: ApplicationType) {
		const { name, description, logo_url, website_url, privacy_url, terms_url } = form;
		fill({ ...blank(type), name, description, logo_url, website_url, privacy_url, terms_url });
	}

	function toggle<T extends string>(list: T[], value: T, on: boolean): T[] {
		return on ? [...list, value] : list.filter((it) => it !== value);
	}

	function payload(): ApplicationInput {
		return {
			...form,
			name: form.name.trim(),
			description: form.description.trim(),
			redirect_uris: lines(redirects),
			post_logout_redirect_uris: lines(logoutRedirects),
			access_token_lifetime: Number(accessMinutes) * 60,
			id_token_lifetime: Number(idMinutes) * 60,
			refresh_token_lifetime: Number(refreshDays) * 86400
		};
	}

	async function saved(result: ApplicationWithSecret) {
		await queryClient.invalidateQueries({ queryKey: keys.applications.all });

		current = result.application;
		fill(asInput(result.application));

		if (result.client_secret) {
			secret = result.client_secret;
		} else {
			open = false;
		}
	}

	const save = createMutation(() => ({
		mutationFn: () =>
			current ? applicationsApi.update(current.id, payload()) : applicationsApi.create(payload()),
		onSuccess: (result: ApplicationWithSecret) => {
			notify.success(
				current ? 'Application saved' : 'Application created',
				result.client_secret ? 'Copy its secret now: it is shown once.' : undefined
			);
			return saved(result);
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this application');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	const rotate = createMutation(() => ({
		mutationFn: () => applicationsApi.rotateSecret(current!.id),
		onSuccess: async (result: ApplicationWithSecret) => {
			notify.success('Secret rotated', 'Copy the new one now: it is shown once.');
			confirmingRotate = false;
			await saved(result);
		},
		onError: (err: unknown) => {
			confirmingRotate = false;
			notify.error(err, 'Could not rotate the secret');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (!editable || tab !== 'settings' || saving || form.name.trim() === '') return;

		saving = true;
		save.mutate();
	}

	function rotateNow() {
		if (!current || saving) return;

		saving = true;
		rotate.mutate();
	}

	const rolesHref = $derived(
		current
			? `${resolve('/admin/(panel)/dashboard/roles')}?tab=application&application=${current.id}`
			: ''
	);

	const offeredGrants = $derived(
		grants.filter((grant) => grant.value !== 'client_credentials' || !kind.public)
	);

	const maskedSecret = $derived(
		current?.has_secret ? `••••••••••••••••${current.secret_hint}` : 'No secret yet'
	);
</script>

{#snippet settings()}
	<fieldset disabled={!editable}>
		<FormSection title="General" description="How the application is named and presented.">
			<Select
				label="Application type"
				value={form.type}
				options={typeOptions}
				readOnly={editing}
				hint={editing
					? 'The type cannot change: copies of the app already in use depend on it.'
					: kind.description}
				onChange={changeType}
			/>

			<Input label="Name" bind:value={form.name} placeholder="Shop" required />

			<Textarea
				label="Description"
				bind:value={form.description}
				rows={2}
				placeholder="What the application is, for other administrators"
			/>

			<FieldGrid>
				<Input
					label="Homepage URL"
					bind:value={form.website_url}
					type="url"
					placeholder="https://shop.example.com"
				/>
				<Input
					label="Logo URL"
					bind:value={form.logo_url}
					type="url"
					placeholder="https://shop.example.com/logo.png"
					hint="https only."
				/>
			</FieldGrid>

			<SwitchField
				label="Is Enabled"
				description="A disabled application cannot sign anyone in."
				bind:checked={form.is_enabled}
			/>
		</FormSection>

		<FormSection
			title="Credentials"
			description={kind.public
				? 'A public client cannot keep a secret, so it has none and must use PKCE.'
				: 'What the application proves who it is with at the token endpoint.'}
		>
			{#if current}
				<Input label="Client ID" value={current.client_id} readOnly copyable />
			{:else}
				<p class="note">
					The client ID{kind.public ? '' : ' and secret'} are made when you create the application.
				</p>
			{/if}

			{#if !kind.public}
				<Select
					label="Client authentication"
					bind:value={form.token_auth_method}
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
								<Button
									size="sm"
									variant="subtle"
									onclick={() => (confirmingRotate = true)}
									disabled={saving}
								>
									<Icon icon={RiRefreshLine} />
									Rotate secret
								</Button>
							</div>
						{/if}
					</div>
				{/if}
			{/if}
		</FormSection>

		{#if !machine}
			<FormSection
				title="Redirects"
				description="Where users may be sent back to. Each URI is matched exactly, with no wildcards."
			>
				<Textarea
					label="Redirect URIs"
					bind:value={redirects}
					rows={3}
					placeholder="https://shop.example.com/auth/callback"
					hint={form.type === 'native'
						? 'One per line. https, a loopback address, or a private-use scheme such as com.example.app:/callback.'
						: 'One per line. https, or http on localhost.'}
				/>

				<Textarea
					label="Post-logout redirect URIs"
					bind:value={logoutRedirects}
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
								checked={form.grant_types.includes(grant.value)}
								onChange={(on) =>
									(form.grant_types = toggle<GrantType>(form.grant_types, grant.value, on))}
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
								checked={form.scopes.includes(scope.value)}
								onChange={(on) => (form.scopes = toggle<Scope>(form.scopes, scope.value, on))}
							/>
						{/each}
					</div>
				</div>

				<SwitchField
					label="Require PKCE"
					description={kind.public
						? 'Always on for a public client.'
						: 'Recommended for every client: it stops a stolen code from being used.'}
					bind:checked={form.require_pkce}
					disabled={kind.public}
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
					bind:value={accessMinutes}
					type="number"
					min="1"
					suffix="minutes"
				/>
				{#if !machine}
					<Input label="ID token" bind:value={idMinutes} type="number" min="1" suffix="minutes" />
					<Input
						label="Refresh token"
						bind:value={refreshDays}
						type="number"
						min="1"
						suffix="days"
					/>
				{/if}
			</div>
		</FormSection>

		{#if !machine}
			<FormSection
				title="Sign-in page"
				description="What users see when this application sends them to sign in."
			>
				<FieldGrid>
					<Input
						label="Terms of service URL"
						bind:value={form.terms_url}
						type="url"
						placeholder="https://shop.example.com/terms"
					/>
					<Input
						label="Privacy policy URL"
						bind:value={form.privacy_url}
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
					bind:checked={form.allow_registration}
				/>

				{#if flowOptions.length > 1}
					<Select
						label="Login flow"
						value={form.login_flow_id || DEFAULT_FLOW}
						onChange={(chosen) => (form.login_flow_id = chosen === DEFAULT_FLOW ? '' : chosen)}
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
							<Icon icon={RiArrowRightLine} size="0.875rem" />
						</a>
					{/if}
				{/snippet}

				<div class="switches">
					<SwitchField
						label="Include roles in tokens"
						description="Put the user's roles in this application into its tokens and userinfo."
						bind:checked={form.assert_roles}
					/>
					<SwitchField
						label="Require a role to sign in"
						description="Only let users who hold at least one of its roles sign in to it."
						bind:checked={form.require_role_assignment}
					/>
				</div>
			</FormSection>
		{/if}
	</fieldset>
{/snippet}

{#snippet actions()}
	{#if secret}
		<Button onclick={() => (open = false)}>Done</Button>
	{:else if editable && tab === 'settings'}
		<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>
		<Button type="submit" loading={saving} disabled={saving || form.name.trim() === ''}>
			{saving ? 'Saving…' : editing ? 'Save changes' : 'Create application'}
		</Button>
	{/if}
{/snippet}

<FullscreenDialog
	bind:open
	actions={secret || (editable && tab === 'settings') ? actions : undefined}
	title={current ? current.name : 'New application'}
	meta={current?.client_id}
	onsubmit={submit}
>
	{#if secret && current}
		<SecretPanel clientId={current.client_id} {secret} />
	{/if}

	{#if current}
		<div class="summary">
			<span class="mark"><Icon icon={kind.icon} size="1.25rem" /></span>
			<div class="summary-text">
				<strong>{kind.label}</strong>
				<span>{kind.public ? 'Public client' : 'Confidential client'}</span>
			</div>
			<Badge tone={current.is_enabled ? 'success' : 'neutral'}>
				{current.is_enabled ? 'Enabled' : 'Disabled'}
			</Badge>
		</div>
	{/if}

	{#if current}
		<Tabs
			tabs={[
				{ value: 'settings', label: 'Settings', icon: RiSettings3Line },
				{ value: 'apis', label: 'API access', icon: RiCodeBoxLine },
				{ value: 'preview', label: 'Token preview', icon: RiShieldCheckLine }
			]}
			bind:value={tab}
			label="Application sections"
		>
			{#snippet panel(value)}
				{#if value === 'settings'}
					{@render settings()}
				{:else if value === 'apis'}
					<ApplicationApis applicationId={current!.id} {editable} />
				{:else}
					<TokenPreviewPanel application={current!} {admin} />
				{/if}
			{/snippet}
		</Tabs>
	{:else}
		{@render settings()}
	{/if}
</FullscreenDialog>

<ConfirmDialog
	bind:open={confirmingRotate}
	title="Rotate the client secret?"
	description="A new secret is made and shown once. The current one stops working at once, so anything still using it can no longer ask for tokens until it is given the new one."
	tone="warning"
	icon={RiRefreshLine}
	confirmLabel="Rotate now"
	busy={saving}
	onConfirm={rotateNow}
/>

<style>
	fieldset {
		min-width: 0;
		margin: 0;
		padding: 0;
		border: none;
	}

	.summary {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		margin-bottom: var(--space-5);
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.mark {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		border-radius: var(--radius-md);
		background: var(--color-secondary-alt);
	}

	.summary-text {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.summary-text span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
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
