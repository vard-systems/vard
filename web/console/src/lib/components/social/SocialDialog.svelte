<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { RiExternalLinkLine } from '#lib/icons.js';
	import {
		socialApi,
		type SocialKind,
		type SocialProvider,
		type SocialProviderInput,
		type SocialSpec,
		type SocialTokenAuth
	} from '#lib/api/index.js';
	import {
		Button,
		ConfirmDialog,
		DangerZone,
		FullscreenDialog,
		FieldGrid,
		FormSection,
		Icon,
		Input,
		Select,
		SwitchField,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { kindOptions, tokenAuthOptions } from './providers';
	import SecretField from './SecretField.svelte';

	type Props = {
		open: boolean;
		provider?: SocialProvider | null;
		kinds: SocialSpec[];
	};

	let { open = $bindable(false), provider = null, kinds }: Props = $props();

	const queryClient = useQueryClient();

	const editing = $derived(provider !== null);

	let kind = $state<SocialKind>('google');
	let slug = $state('');
	let name = $state('');
	let clientID = $state('');
	let clientSecret = $state('');
	let teamID = $state('');
	let keyID = $state('');
	let privateKey = $state('');
	let scopes = $state('');
	let tokenAuth = $state<SocialTokenAuth | ''>('');
	let authorizeURL = $state('');
	let tokenURL = $state('');
	let userInfoURL = $state('');
	let enabled = $state(true);
	let linkVerifiedEmails = $state(true);
	let allowRegistration = $state(true);

	let saving = $state(false);
	let confirmingDelete = $state(false);

	const spec = $derived(kinds.find((one) => one.kind === kind));

	const signs = $derived(spec?.signed_secret ?? false);
	const custom = $derived(spec?.custom ?? false);

	const ready = $derived(
		name.trim() !== '' &&
			clientID.trim() !== '' &&
			(signs
				? (privateKey.trim() !== '' || (provider?.has_private_key ?? false)) &&
					teamID.trim() !== '' &&
					keyID.trim() !== ''
				: clientSecret.trim() !== '' || (provider?.has_client_secret ?? false)) &&
			(!custom || (authorizeURL.trim() !== '' && tokenURL.trim() !== ''))
	);

	$effect(() => {
		if (!open) return;

		confirmingDelete = false;
		clientSecret = '';
		privateKey = '';

		if (provider) {
			kind = provider.kind;
			slug = provider.slug;
			name = provider.name;
			clientID = provider.client_id;
			teamID = provider.team_id;
			keyID = provider.key_id;
			scopes = provider.scopes.join(' ');
			tokenAuth = provider.token_auth;
			authorizeURL = provider.authorize_url;
			tokenURL = provider.token_url;
			userInfoURL = provider.userinfo_url;
			enabled = provider.is_enabled;
			linkVerifiedEmails = provider.link_verified_emails;
			allowRegistration = provider.allow_registration;
			return;
		}

		fillFromKind(kinds[0]?.kind ?? 'google');
		enabled = true;
		linkVerifiedEmails = true;
		allowRegistration = true;
	});

	function fillFromKind(chosen: SocialKind) {
		kind = chosen;

		const known = kinds.find((one) => one.kind === chosen);
		slug = chosen;
		name = known?.label ?? chosen;
		scopes = (known?.scopes ?? []).join(' ');
		tokenAuth = '';
		authorizeURL = known?.custom ? '' : (known?.authorize_url ?? '');
		tokenURL = known?.custom ? '' : (known?.token_url ?? '');
		userInfoURL = known?.custom ? '' : (known?.userinfo_url ?? '');
		clientID = '';
		clientSecret = '';
		teamID = '';
		keyID = '';
		privateKey = '';
	}

	function input(): SocialProviderInput {
		const body: SocialProviderInput = {
			name: name.trim(),
			client_id: clientID.trim(),
			scopes: scopes.split(/\s+/).filter(Boolean),
			is_enabled: enabled,
			link_verified_emails: linkVerifiedEmails,
			allow_registration: allowRegistration
		};

		// A secret is only ever sent when one was typed: what is stored
		// cannot be read back, and an empty field means "leave it".
		if (clientSecret.trim() !== '') body.client_secret = clientSecret.trim();
		if (privateKey.trim() !== '') body.private_key = privateKey.trim();

		if (signs) {
			body.team_id = teamID.trim();
			body.key_id = keyID.trim();
		}

		if (custom) {
			body.authorize_url = authorizeURL.trim();
			body.token_url = tokenURL.trim();
			body.userinfo_url = userInfoURL.trim();
			body.token_auth = tokenAuth;
		}

		if (!editing) {
			body.kind = kind;
			body.slug = slug.trim().toLowerCase();
		}

		return body;
	}

	const save = createMutation(() => ({
		mutationFn: () =>
			provider ? socialApi.update(provider.id, input()) : socialApi.create(input()),
		onSuccess: async () => {
			notify.success(provider ? 'Provider saved' : 'Provider added');
			open = false;
			await queryClient.invalidateQueries({ queryKey: keys.social.all });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this provider');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	const remove = createMutation(() => ({
		mutationFn: () => socialApi.remove(provider?.id ?? ''),
		onSuccess: async () => {
			notify.success('Provider removed');
			open = false;
			await queryClient.invalidateQueries({ queryKey: keys.social.all });
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not remove this provider');
			saving = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (!ready || saving) return;

		saving = true;
		save.mutate();
	}
</script>

<FullscreenDialog
	bind:open
	title={editing ? provider!.name : 'Add a provider'}
	description={editing
		? 'What this server sends to the provider, and what it may do here.'
		: 'Register this server with a provider, then paste what it gave you here.'}
	meta={editing ? provider!.slug : undefined}
	onsubmit={submit}
>
	<FormSection
		title="Provider"
		description={editing
			? 'The kind and the identifier are in the address registered with the provider, so they stay as they are.'
			: 'Choosing one fills in everything about it this server already knows.'}
	>
		{#if editing}
			<Input label="Kind" value={spec?.label ?? kind} readOnly />
		{:else}
			<Select
				label="Kind"
				value={kind}
				options={kindOptions(kinds)}
				onChange={(chosen) => fillFromKind(chosen)}
			/>
		{/if}

		<FieldGrid>
			<Input label="Name" bind:value={name} required hint="What the button says." />
			<Input
				label="Identifier"
				bind:value={slug}
				readOnly={editing}
				required
				hint="In the callback address."
			/>
		</FieldGrid>

		{#if spec?.docs}
			<p class="note">
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={spec.docs} target="_blank" rel="noopener noreferrer">
					Register this server with {spec.label}
					<Icon icon={RiExternalLinkLine} size="0.8rem" />
				</a>
			</p>
		{/if}
	</FormSection>

	{#if editing}
		<FormSection
			title="Redirect URI"
			description="What the provider has to be told to send people back to. It has to match exactly."
		>
			<Input label="Callback URL" value={provider!.callback_url} readOnly copyable />
		</FormSection>
	{/if}

	<FormSection
		title="Credentials"
		description="What the provider gave you when you registered, and what to ask it for."
	>
		<Input label="Client ID" bind:value={clientID} required copyable={editing} />

		{#if signs}
			<FieldGrid>
				<Input label="Team ID" bind:value={teamID} required copyable={editing} />
				<Input label="Key ID" bind:value={keyID} required copyable={editing} />
			</FieldGrid>

			<SecretField
				label="Signing key"
				providerId={provider?.id}
				stored={provider?.has_private_key ?? false}
				bind:value={privateKey}
				multiline
				hint="The .p8 file from the developer account."
			/>
		{:else}
			<SecretField
				label="Client secret"
				providerId={provider?.id}
				stored={provider?.has_client_secret ?? false}
				bind:value={clientSecret}
			/>
		{/if}

		<Input
			label="Scopes"
			bind:value={scopes}
			placeholder="openid email profile"
			hint="Separated by spaces."
		/>
	</FormSection>

	{#if custom}
		<FormSection
			title="Endpoints"
			description="Where this provider signs people in. Its discovery document lists all three."
		>
			<Input label="Authorization URL" bind:value={authorizeURL} required />
			<Input label="Token URL" bind:value={tokenURL} required />
			<Input
				label="Userinfo URL"
				bind:value={userInfoURL}
				hint="OpenID Connect providers may leave this out; the id_token carries the same claims."
			/>
			<Select label="Client authentication" bind:value={tokenAuth} options={tokenAuthOptions} />
		</FormSection>
	{/if}

	<FormSection
		title="What it may do here"
		description="Whether it is offered, and what signing in with it can lead to."
	>
		<SwitchField
			label="Offer this provider"
			description="Show its button on the sign-in pages."
			bind:checked={enabled}
		/>

		<SwitchField
			label="Link to existing accounts"
			description="Sign someone in to the account that already has their address, when the provider says it has verified it. Off, they have to sign in and connect it themselves."
			bind:checked={linkVerifiedEmails}
		/>

		<SwitchField
			label="Create accounts"
			description="Let somebody with no account here make one by signing in with this provider."
			bind:checked={allowRegistration}
		/>
	</FormSection>

	{#if editing}
		<DangerZone
			title="Remove this provider"
			description="The accounts that signed in with it stay; this way in goes."
			onclick={() => (confirmingDelete = true)}
		/>
	{/if}

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)}>Cancel</Button>
		<Button type="submit" loading={saving} disabled={!ready || saving}>
			{editing ? 'Save changes' : 'Add provider'}
		</Button>
	{/snippet}
</FullscreenDialog>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Remove ${provider?.name ?? 'this provider'}?`}
	description={(provider?.identities ?? 0) > 0
		? `${provider?.identities} ${provider?.identities === 1 ? 'person signs' : 'people sign'} in with this. Their accounts stay; this way in goes.`
		: 'Nobody signs in with this yet.'}
	confirmLabel="Remove provider"
	busy={remove.isPending}
	onConfirm={() => remove.mutate()}
/>

<style>
	.note {
		margin: 0;
		font-size: var(--text-sm);
	}

	.note a {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--color-text-hint);
	}

	.note a:hover {
		color: var(--color-text);
	}
</style>
