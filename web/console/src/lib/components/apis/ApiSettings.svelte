<script lang="ts">
	import { enhance } from '$app/forms';
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { apisApi, type API, type SigningAlgorithm } from '#lib/api/index.js';
	import {
		Button,
		ConfirmDialog,
		DangerZone,
		FieldGrid,
		FormSection,
		Input,
		Note,
		Select,
		SwitchField,
		Textarea,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { submit } from '#lib/forms.js';
	import { algorithms, apiInput, roleBasedAccess } from './scopes';

	type Props = {
		api: API;
		editable: boolean;
	};

	let { api, editable }: Props = $props();

	const queryClient = useQueryClient();

	let name = $state('');
	let description = $state('');
	let enforceRoles = $state(true);
	let algorithm = $state<SigningAlgorithm>('RS256');
	let lifetimeMinutes = $state('');
	let offlineAccess = $state(false);

	let saving = $state(false);

	let confirmingDelete = $state(false);
	let deleting = $state(false);

	function fill(from: API) {
		name = from.name;
		description = from.description;
		enforceRoles = from.enforce_roles;
		algorithm = from.signing_algorithm;
		lifetimeMinutes = from.token_lifetime > 0 ? String(Math.round(from.token_lifetime / 60)) : '';
		offlineAccess = from.allow_offline_access;
	}

	// Filled once per API: a refetch in the background leaves what is being
	// typed alone.
	const id = $derived(api.id);
	$effect(() => {
		void id;
		untrack(() => fill(api));
	});

	const lifetime = $derived(lifetimeMinutes.trim() === '' ? 0 : Number(lifetimeMinutes) * 60);

	const dirty = $derived(
		name.trim() !== api.name ||
			description.trim() !== api.description ||
			enforceRoles !== api.enforce_roles ||
			algorithm !== api.signing_algorithm ||
			lifetime !== api.token_lifetime ||
			offlineAccess !== api.allow_offline_access
	);

	const save = submit<Awaited<ReturnType<typeof apisApi.update>>>({
		input: () => [
			api.id,
			apiInput(api, {
				name: name.trim(),
				description: description.trim(),
				enforce_roles: enforceRoles,
				signing_algorithm: algorithm,
				token_lifetime: lifetime,
				allow_offline_access: offlineAccess
			})
		],
		when: () => !(!editable || !dirty || name.trim() === ''),
		pending: (busy) => (saving = busy),
		done: async (result) => {
			queryClient.setQueryData(keys.apis.one(api.id), result.api);
			fill(result.api);
			notify.success('Settings saved');
			await queryClient.invalidateQueries({ queryKey: keys.apis.all });
		},
		failed: (err) => notify.error(err, 'Could not save this API')
	});

	const remove = createMutation(() => ({
		mutationFn: () => apisApi.remove(api.id),
		onSuccess: async () => {
			notify.success(`${api.name} deleted`);
			await goto(resolve('/(panel)/apis'));
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.apis.all }),
				queryClient.invalidateQueries({ queryKey: keys.roles.all }),
				queryClient.invalidateQueries({ queryKey: keys.applications.all })
			]);
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not delete this API');
			deleting = false;
			confirmingDelete = false;
		}
	}));
</script>

<form method="POST" action="/actions/apis?/update" use:enhance={save}>
	<fieldset disabled={!editable}>
		<FormSection title="General" description="How the API is named and described in this panel.">
			<Input label="Name" bind:value={name} required />

			<Input
				label="Identifier"
				icon="ri-links-line"
				value={api.identifier}
				readOnly
				copyable
				hint="The audience of its tokens. It cannot change: every service validating them compares against it. Register a new API to use another."
			/>

			<Textarea label="Description" bind:value={description} rows={2} />
		</FormSection>

		<FormSection title="Access" description="What decides the scopes a user's token receives.">
			<SwitchField
				label={roleBasedAccess.label}
				description={roleBasedAccess.description}
				bind:checked={enforceRoles}
			/>

			<Note>
				Either way, the API must still enforce permissions: it checks the scope an endpoint needs is
				in the token.
			</Note>
		</FormSection>

		<FormSection
			title="Tokens"
			description="How access tokens for this API are signed and how long they last."
		>
			<FieldGrid>
				<Select
					label="Signing algorithm"
					bind:value={algorithm}
					options={algorithms}
					readOnly={!editable}
				/>

				<Input
					label="Access token lifetime"
					bind:value={lifetimeMinutes}
					type="number"
					min="1"
					max="1440"
					suffix="minutes"
					placeholder="Application's"
					hint="Empty uses each application's lifetime."
				/>
			</FieldGrid>

			<SwitchField
				label="Allow refresh tokens"
				description="Let applications with the refresh_token grant get a refresh token for this API by asking for offline_access."
				bind:checked={offlineAccess}
			/>
		</FormSection>
	</fieldset>

	{#if editable}
		<div class="actions">
			<Button variant="subtle" onclick={() => fill(api)} disabled={!dirty || saving}>Discard</Button
			>
			<Button type="submit" loading={saving} disabled={!dirty || saving || name.trim() === ''}>
				{saving ? 'Saving…' : 'Save changes'}
			</Button>
		</div>

		{#if !api.system}
			<DangerZone
				title="Delete this API"
				description={`${
					api.application_count > 0
						? `${api.application_count} ${api.application_count === 1 ? 'application loses' : 'applications lose'} access, and roles lose its scopes.`
						: 'Its scopes are removed from every role.'
				} This cannot be undone.`}
				label="Delete"
				onclick={() => (confirmingDelete = true)}
			/>
		{/if}
	{/if}
</form>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Delete ${api.name}?`}
	description={api.application_count > 0
		? `${api.application_count} ${api.application_count === 1 ? 'application loses' : 'applications lose'} access, and roles lose its scopes. This cannot be undone.`
		: 'Its scopes are removed from every role. This cannot be undone.'}
	confirmLabel="Delete API"
	busy={deleting}
	onConfirm={() => {
		deleting = true;
		remove.mutate();
	}}
/>

<style>
	form {
		max-width: 44rem;
	}

	fieldset {
		min-width: 0;
		margin: 0;
		padding: 0;
		border: none;
	}

	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
		margin-top: var(--space-4);
	}
</style>
