<script lang="ts">
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { apisApi, type SigningAlgorithm } from '#lib/api/index.js';
	import {
		Button,
		FullscreenDialog,
		FormSection,
		Input,
		Select,
		SwitchField,
		Textarea,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import ScopeEditor from './ScopeEditor.svelte';
	import {
		algorithms,
		roleBasedAccess,
		scopeInput,
		scopeProblem,
		scopeRow,
		type ScopeRow
	} from './scopes';

	type Props = {
		open: boolean;
	};

	let { open = $bindable(false) }: Props = $props();

	const queryClient = useQueryClient();

	let name = $state('');
	let identifier = $state('');
	let description = $state('');
	let enforceRoles = $state(true);
	let rows = $state<ScopeRow[]>([]);

	let algorithm = $state<SigningAlgorithm>('RS256');
	let lifetimeMinutes = $state('');
	let offlineAccess = $state(false);

	let saving = $state(false);

	const ready = $derived(
		name.trim() !== '' &&
			identifier.trim() !== '' &&
			rows.every((row) => scopeProblem(rows, row) === undefined)
	);

	$effect(() => {
		if (!open) return;

		untrack(() => {
			name = '';
			identifier = '';
			description = '';
			enforceRoles = true;
			rows = [scopeRow()];
			algorithm = 'RS256';
			lifetimeMinutes = '';
			offlineAccess = false;
		});
	});

	const create = createMutation(() => ({
		mutationFn: () =>
			apisApi.create({
				name: name.trim(),
				identifier: identifier.trim(),
				description: description.trim(),
				enforce_roles: enforceRoles,
				signing_algorithm: algorithm,
				token_lifetime: lifetimeMinutes.trim() === '' ? 0 : Number(lifetimeMinutes) * 60,
				allow_offline_access: offlineAccess,
				scopes: scopeInput(rows)
			}),
		onSuccess: async (result) => {
			notify.success(`${result.api.name} created`);
			await queryClient.invalidateQueries({ queryKey: keys.apis.all });
			open = false;
			await goto(resolve('/(panel)/apis/[id]', { id: result.api.id }));
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not create this API');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (saving || !ready) return;

		saving = true;
		create.mutate();
	}
</script>

<FullscreenDialog bind:open title="New API" onsubmit={submit}>
	<FormSection
		title="API"
		description="A protected resource applications request access tokens for. Its identifier is the audience of those tokens."
	>
		<Input label="Name" bind:value={name} placeholder="Marketplace API" required />

		<Input
			label="Identifier"
			icon="ri-links-line"
			bind:value={identifier}
			placeholder="https://api.example.com"
			hint="The aud claim of its access tokens. Unique, and it cannot change later: every service validating tokens compares against it. A URL is usual; it does not have to resolve."
			autocapitalize="none"
			spellcheck={false}
			required
		/>

		<Textarea
			label="Description"
			bind:value={description}
			rows={2}
			placeholder="What the API does, for other administrators"
		/>

		<SwitchField
			label={roleBasedAccess.label}
			description={roleBasedAccess.description}
			bind:checked={enforceRoles}
		/>
	</FormSection>

	<FormSection
		title="Scopes"
		description="The permissions the API exposes. Applications are allowed some of them; roles grant them to users. A default scope is added to every token for the API."
	>
		<ScopeEditor bind:rows editable />
	</FormSection>

	<FormSection
		title="Tokens"
		description="How its access tokens are signed and how long they last. The defaults suit most APIs."
	>
		<Select
			label="Token signing algorithm"
			bind:value={algorithm}
			options={algorithms}
			hint="Resource servers validate tokens with the public keys at this server's JWKS URI."
		/>

		<Input
			label="Access token lifetime"
			bind:value={lifetimeMinutes}
			type="number"
			min="1"
			max="1440"
			suffix="minutes"
			placeholder="Application's"
			hint="Leave empty to use each application's access token lifetime."
		/>

		<SwitchField
			label="Allow refresh tokens"
			description="Let applications get a refresh token alongside an access token for this API."
			bind:checked={offlineAccess}
		/>
	</FormSection>

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>
		<Button type="submit" loading={saving} disabled={saving || !ready}>
			{saving ? 'Creating…' : 'Create API'}
		</Button>
	{/snippet}
</FullscreenDialog>
