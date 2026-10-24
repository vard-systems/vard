<script lang="ts">
	import { createMutation, createQuery } from '@tanstack/svelte-query';
	import {
		applicationsApi,
		type Admin,
		type Application,
		type TokenPreview,
		type UserRecord
	} from '#lib/api/index.js';
	import {
		Button,
		SegmentedControl,
		Select,
		type SelectOption,
		notify
	} from '#lib/components/ui/index.js';
	import { can } from '#lib/permissions.js';
	import { applicationAccessOptions } from '#lib/query/index.js';
	import { userGrants } from './applications';
	import ScopePicker from './token-preview/ScopePicker.svelte';
	import TokenVerdict from './token-preview/TokenVerdict.svelte';
	import UserPicker from './token-preview/UserPicker.svelte';

	type Props = {
		application: Application;
		admin: Admin;
	};

	let { application, admin }: Props = $props();

	const NO_AUDIENCE = 'none';

	const forUsers = $derived(userGrants.some((grant) => application.grant_types.includes(grant)));
	const forItself = $derived(application.grant_types.includes('client_credentials'));

	let picked = $state<'user' | 'client'>('user');
	let user = $state<UserRecord | null>(null);
	let audience = $state(NO_AUDIENCE);
	let chosen = $state<string[]>([]);
	let extra = $state('');

	let preview = $state<TokenPreview | null>(null);
	let running = $state(false);

	/** An application that only acts for itself has no users to ask about. */
	const subject = $derived(!forUsers && forItself ? 'client' : picked);

	const subjects = $derived([
		{ value: 'user' as const, label: 'A user', disabled: !forUsers },
		{ value: 'client' as const, label: 'The application itself', disabled: !forItself }
	]);

	const access = createQuery(() => applicationAccessOptions(application.id));

	const audienceOptions = $derived<SelectOption[]>([
		{
			value: NO_AUDIENCE,
			label: 'No API',
			description: 'Sign-in only: an ID token and userinfo'
		},
		...(access.data ?? []).map((api) => ({
			value: api.identifier,
			label: api.name,
			description: api.authorized ? api.identifier : `${api.identifier} · not authorised`
		}))
	]);

	const audienceApi = $derived((access.data ?? []).find((api) => api.identifier === audience));

	const offered = $derived([
		...(subject === 'user' ? application.scopes : []),
		...(audienceApi?.scopes.map((scope) => scope.name) ?? [])
	]);

	/** Another API's scopes no longer apply; the application's own still do. */
	function setAudience(value: string) {
		audience = value;
		chosen = chosen.filter((scope) => application.scopes.includes(scope as never));
	}

	const scopeParam = $derived(
		[...chosen, ...extra.split(/\s+/)]
			.filter((scope, i, all) => scope && all.indexOf(scope) === i)
			.join(' ')
	);

	const canRun = $derived(!running && (subject === 'client' || user !== null));

	const evaluate = createMutation(() => ({
		mutationFn: () =>
			applicationsApi.previewToken(application.id, {
				user_id: subject === 'user' ? (user?.id ?? null) : null,
				audience: audience === NO_AUDIENCE ? '' : audience,
				scope: scopeParam
			}),
		onSuccess: (result: { preview: TokenPreview }) => {
			preview = result.preview;
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not evaluate this token request');
		},
		onSettled: () => {
			running = false;
		}
	}));

	function run() {
		if (!canRun) return;

		running = true;
		evaluate.mutate();
	}
</script>

<p class="lead">
	See exactly what a token for this application would carry, without issuing one. It runs the same
	rules the token endpoint uses.
</p>

<div class="form">
	<div class="field">
		<span class="label">Token for</span>
		<div class="subject">
			<SegmentedControl
				label="Token for"
				options={subjects}
				value={subject}
				onChange={(value) => (picked = value)}
			/>
		</div>
	</div>

	{#if subject === 'user'}
		<UserPicker bind:user allowed={can(admin, 'users.read')} />
	{/if}

	<Select label="Audience" bind:value={audience} options={audienceOptions} onChange={setAudience} />

	<ScopePicker {offered} bind:chosen bind:extra param={scopeParam} onenter={run} />

	<div class="run">
		<Button onclick={run} disabled={!canRun} loading={running}>
			<i class="ri-shield-check-line" aria-hidden="true"></i>
			Evaluate
		</Button>
	</div>
</div>

{#if preview}
	<TokenVerdict {preview} />
{/if}

<style>
	.lead {
		margin: 0 0 var(--space-4);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.45;
	}

	.form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		padding: var(--space-4);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.label {
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.subject {
		align-self: flex-start;
	}

	.run {
		display: flex;
		justify-content: flex-end;
	}
</style>
