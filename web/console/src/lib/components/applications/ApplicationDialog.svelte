<script module lang="ts">
	import type { Tab } from '#lib/components/ui/index.js';

	const tabs: Tab[] = [
		{ value: 'settings', label: 'Settings', icon: 'ri-settings-3-line' },
		{ value: 'apis', label: 'API access', icon: 'ri-code-box-line' },
		{ value: 'preview', label: 'Token preview', icon: 'ri-shield-check-line' }
	];
</script>

<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import {
		applicationsApi,
		type Admin,
		type Application,
		type ApplicationWithSecret
	} from '#lib/api/index.js';
	import {
		Button,
		ConfirmDialog,
		FullscreenDialog,
		Tabs,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import ApplicationApis from './ApplicationApis.svelte';
	import ApplicationSettings from './ApplicationSettings.svelte';
	import ApplicationSummary from './ApplicationSummary.svelte';
	import { ApplicationForm } from './form.svelte';
	import SecretPanel from './SecretPanel.svelte';
	import TokenPreviewPanel from './TokenPreviewPanel.svelte';

	type Props = {
		application: Application | null;
		editable: boolean;
		admin: Admin;
		open: boolean;
	};

	let { application, editable, admin, open = $bindable(false) }: Props = $props();

	const queryClient = useQueryClient();

	const form = new ApplicationForm();

	/** The application as last saved; null until a new one is created. */
	let current = $state<Application | null>(null);
	let tab = $state('settings');
	/** A client secret just made: shown once, here, and never again. */
	let secret = $state('');
	let saving = $state(false);
	let confirmingRotate = $state(false);

	const saveable = $derived(editable && tab === 'settings');

	$effect(() => {
		if (!open) {
			tab = 'settings';
			return;
		}

		current = application;
		secret = '';
		confirmingRotate = false;
		form.fill(application);
	});

	async function saved(result: ApplicationWithSecret) {
		await queryClient.invalidateQueries({ queryKey: keys.applications.all });

		current = result.application;
		form.fill(result.application);

		if (result.client_secret) {
			secret = result.client_secret;
		} else {
			open = false;
		}
	}

	const save = createMutation(() => ({
		mutationFn: () =>
			current
				? applicationsApi.update(current.id, form.toInput())
				: applicationsApi.create(form.toInput()),
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

		if (!saveable || saving || !form.ready) return;

		saving = true;
		save.mutate();
	}

	function rotateNow() {
		if (!current || saving) return;

		saving = true;
		rotate.mutate();
	}
</script>

{#snippet settings()}
	<ApplicationSettings
		{form}
		{current}
		{editable}
		{saving}
		onrotate={() => (confirmingRotate = true)}
	/>
{/snippet}

{#snippet actions()}
	{#if secret}
		<Button onclick={() => (open = false)}>Done</Button>
	{:else if saveable}
		<Button variant="subtle" onclick={() => (open = false)} disabled={saving}>Cancel</Button>
		<Button type="submit" loading={saving} disabled={saving || !form.ready}>
			{saving ? 'Saving…' : current ? 'Save changes' : 'Create application'}
		</Button>
	{/if}
{/snippet}

<FullscreenDialog
	bind:open
	actions={secret || saveable ? actions : undefined}
	title={current ? current.name : 'New application'}
	meta={current?.client_id}
	onsubmit={submit}
>
	{#if current}
		{#if secret}
			<SecretPanel clientId={current.client_id} {secret} />
		{/if}

		<ApplicationSummary application={current} />

		<Tabs {tabs} bind:value={tab} label="Application sections">
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
	icon="ri-refresh-line"
	confirmLabel="Rotate now"
	busy={saving}
	onConfirm={rotateNow}
/>
