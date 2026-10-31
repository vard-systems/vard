<script module lang="ts">
	import type { Tab } from '#lib/components/ui/index.js';

	const tabs: Tab[] = [
		{ value: 'connection', label: 'Connection', icon: 'ri-settings-3-line' },
		{ value: 'provider', label: 'Service provider', icon: 'ri-server-line' },
		{ value: 'signin', label: 'Domains & sign-in', icon: 'ri-links-line' },
		{ value: 'users', label: 'Users & roles', icon: 'ri-group-line' }
	];
</script>

<script lang="ts">
	import { submit } from '#lib/forms.js';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ssoApi, type Role, type SSOConnection } from '#lib/api/index.js';
	import {
		Button,
		ConfirmDialog,
		FullscreenDialog,
		Tabs,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { SSOForm, type SSOTab } from './form.svelte';
	import SSOConnectionTab from './SSOConnectionTab.svelte';
	import SSOServiceProviderTab from './SSOServiceProviderTab.svelte';
	import SSOSignInTab from './SSOSignInTab.svelte';
	import SSOUsersTab from './SSOUsersTab.svelte';

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

	const form = new SSOForm();

	/** The connection as last saved; null until a new one is created. */
	let current = $state<SSOConnection | null>(null);
	let tab = $state<SSOTab>('connection');
	let saving = $state(false);
	let confirmingDelete = $state(false);

	$effect(() => {
		if (!open) return;

		current = connection;
		form.fill(connection);
		tab = 'connection';
		confirmingDelete = false;
	});

	const save = submit<Awaited<ReturnType<typeof ssoApi.create>>>({
		input: () => {
			const body = form.toInput(current === null);
			return current ? [current.id, body] : [body];
		},
		when: () => !form.missing && !saving && canWrite,
		pending: (busy) => (saving = busy),
		done: async ({ connection: saved }) => {
			const created = current === null;
			notify.success(created ? 'Connection created' : 'Connection saved');
			await queryClient.invalidateQueries({ queryKey: keys.sso.all });

			// A new connection stays open on what its provider needs to be told.
			if (created) {
				current = saved;
				form.fill(saved);
				tab = 'provider';
				return;
			}

			open = false;
		},
		failed: (err) => notify.error(err, 'Could not save this connection')
	});

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
</script>

{#snippet status()}
	{#if form.missing}
		{@const missing = form.missing}
		<button type="button" class="missing" onclick={() => (tab = missing.tab)}>
			{missing.text}
		</button>
	{/if}
{/snippet}

<FullscreenDialog
	bind:open
	status={canWrite && form.missing ? status : undefined}
	title={current ? current.name : 'New connection'}
	description={current
		? 'Where it signs people in, for which domains, and what they become here.'
		: "An organisation's identity provider. It starts off, so it can be set up and tested before anybody signs in through it."}
	meta={current?.slug}
	action={current ? '/actions/sso?/update' : '/actions/sso?/create'}
	onsubmit={save}
>
	<Tabs {tabs} bind:value={() => tab, (value) => (tab = value as SSOTab)} label="SSO integrations">
		{#snippet panel(value)}
			<div class="panel">
				{#if value === 'connection'}
					<SSOConnectionTab
						{form}
						{current}
						{canWrite}
						onremove={() => (confirmingDelete = true)}
					/>
				{:else if value === 'provider'}
					<SSOServiceProviderTab {current} {canWrite} onrefreshed={(saved) => (current = saved)} />
				{:else if value === 'signin'}
					<SSOSignInTab {form} readOnly={!canWrite} connectionId={current?.id} />
				{:else}
					<SSOUsersTab {form} {roles} {canReadRoles} readOnly={!canWrite} />
				{/if}
			</div>
		{/snippet}
	</Tabs>

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)}>Cancel</Button>
		{#if canWrite}
			<Button type="submit" loading={saving} disabled={form.missing !== null || saving}>
				<i class="ri-check-line" aria-hidden="true"></i>
				{current ? 'Save changes' : 'Create connection'}
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
