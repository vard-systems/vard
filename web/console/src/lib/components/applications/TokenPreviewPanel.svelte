<script lang="ts">
	import { createMutation, createQuery } from '@tanstack/svelte-query';
	import {
		RiCheckLine,
		RiCloseLine,
		RiSearchLine,
		RiShieldCheckLine,
		RiUserLine
	} from 'svelte-remixicon';
	import {
		applicationsApi,
		type Admin,
		type Application,
		type TokenPreview,
		type UserRecord
	} from '$lib/api';
	import {
		Button,
		CodeBlock,
		Icon,
		IconButton,
		Note,
		Select,
		type SelectOption,
		notify
	} from '$lib/components/ui';
	import { can } from '$lib/permissions';
	import { applicationAccessOptions, userSearchOptions } from '$lib/query';

	type Props = {
		application: Application;
		admin: Admin;
	};

	let { application, admin }: Props = $props();

	const NO_AUDIENCE = 'none';

	/** Who the token is for: a user signing in, or the application itself. */
	const forUsers = $derived(application.grant_types.includes('authorization_code'));
	const forItself = $derived(application.grant_types.includes('client_credentials'));

	let subject = $state<'user' | 'client'>('user');
	let user = $state<UserRecord | null>(null);
	let userSearch = $state('');
	let audience = $state(NO_AUDIENCE);
	let chosen = $state<string[]>([]);
	let extra = $state('');

	let preview = $state<TokenPreview | null>(null);
	let running = $state(false);

	$effect(() => {
		// A machine-to-machine application has no users to sign in.
		if (!forUsers && forItself) subject = 'client';
	});

	const mayPickUsers = $derived(can(admin, 'users.read'));

	/** The users matching what is typed, a handful at a time. */
	const users = createQuery(() => ({
		...userSearchOptions(userSearch.trim()),
		enabled: subject === 'user' && mayPickUsers && user === null
	}));

	/** The APIs the application is authorised for, which are the audiences it
	    can ask for. */
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

	/** The scopes worth offering: the OpenID Connect scopes the application
	    may use, for a user; and every scope of the chosen API, allowed or not,
	    so a refusal can be seen. */
	const offered = $derived([
		...(subject === 'user' ? application.scopes : []),
		...(audienceApi?.scopes.map((scope) => scope.name) ?? [])
	]);

	function toggle(scope: string) {
		chosen = chosen.includes(scope) ? chosen.filter((it) => it !== scope) : [...chosen, scope];
	}

	function setAudience(value: string) {
		audience = value;
		// Keep the OpenID scopes, drop the previous API's.
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

	/** Enter in a text box would submit the application form around the tab. */
	function stayPut(event: KeyboardEvent) {
		if (event.key === 'Enter') {
			event.preventDefault();
			run();
		}
	}

	/** The parts of the tokens there would be, in the order they are read. */
	function blocks(result: TokenPreview): { title: string; value: unknown }[] {
		return [
			{ title: 'Access token header', value: result.access_token_header },
			{ title: 'Access token claims', value: result.access_token },
			{ title: 'ID token claims', value: result.id_token }
		].filter((block) => block.value);
	}

	function json(value: unknown): string {
		return JSON.stringify(value, null, 2);
	}
</script>

<p class="lead">
	See exactly what a token for this application would carry, without issuing one. It runs the same
	rules the token endpoint uses.
</p>

<div class="form">
	<div class="field">
		<span class="label">Token for</span>
		<div class="segments" role="radiogroup" aria-label="Token for">
			<button
				type="button"
				role="radio"
				aria-checked={subject === 'user'}
				class:on={subject === 'user'}
				disabled={!forUsers}
				onclick={() => (subject = 'user')}
			>
				A user
			</button>
			<button
				type="button"
				role="radio"
				aria-checked={subject === 'client'}
				class:on={subject === 'client'}
				disabled={!forItself}
				onclick={() => (subject = 'client')}
			>
				The application itself
			</button>
		</div>
	</div>

	{#if subject === 'user'}
		<div class="field">
			<span class="label">User</span>
			{#if !mayPickUsers}
				<Note>Your roles do not include seeing users.</Note>
			{:else if user}
				<div class="picked">
					<Icon icon={RiUserLine} />
					<span>{user.email}</span>
					<IconButton
						icon={RiCloseLine}
						label="Choose another user"
						size="sm"
						onclick={() => (user = null)}
					/>
				</div>
			{:else}
				<div class="user-search">
					<label class="search">
						<Icon icon={RiSearchLine} />
						<input
							id="token-preview-user-search"
							name="token-preview-user-search"
							type="search"
							placeholder="Search users by email or any field…"
							bind:value={userSearch}
							onkeydown={(event) => event.key === 'Enter' && event.preventDefault()}
							aria-label="Search users"
						/>
					</label>
					<ul class="results">
						{#each users.data ?? [] as found (found.id)}
							<li>
								<button type="button" onclick={() => (user = found)}>
									<Icon icon={RiUserLine} size="0.9375rem" />
									{found.email}
								</button>
							</li>
						{:else}
							<li class="muted">{users.isPending ? 'Searching…' : 'No users match this.'}</li>
						{/each}
					</ul>
				</div>
			{/if}
		</div>
	{/if}

	<Select label="Audience" bind:value={audience} options={audienceOptions} onChange={setAudience} />

	<div class="field">
		<span class="label">Scopes</span>
		{#if offered.length > 0}
			<div class="chips">
				{#each offered as scope (scope)}
					<button
						type="button"
						class:on={chosen.includes(scope)}
						aria-pressed={chosen.includes(scope)}
						onclick={() => toggle(scope)}
					>
						{#if chosen.includes(scope)}
							<Icon icon={RiCheckLine} size="0.8125rem" />
						{/if}
						{scope}
					</button>
				{/each}
			</div>
		{/if}
		<input
			id="token-preview-extra-scopes"
			name="token-preview-extra-scopes"
			class="extra"
			bind:value={extra}
			placeholder="Other scopes, separated by spaces"
			onkeydown={stayPut}
			aria-label="Other scopes"
			autocapitalize="none"
			spellcheck="false"
		/>
		<code class="param">scope={scopeParam || '(none)'}</code>
	</div>

	<div class="run">
		<Button onclick={run} disabled={!canRun} loading={running}>
			<Icon icon={RiShieldCheckLine} />
			Evaluate
		</Button>
	</div>
</div>

{#if preview}
	<div class="result">
		<div class="verdict" class:issued={preview.issued}>
			<Icon icon={preview.issued ? RiCheckLine : RiCloseLine} size="1.125rem" />
			<div>
				<strong>{preview.issued ? 'A token would be issued' : 'No token would be issued'}</strong>
				{#if preview.reason}
					<p>{preview.reason}</p>
				{:else if preview.issued}
					<p>
						{preview.decisions.filter((it) => it.granted).length} of {preview.decisions.length}
						requested {preview.decisions.length === 1 ? 'scope' : 'scopes'} granted.
					</p>
				{/if}
			</div>
		</div>

		{#if preview.decisions.length > 0}
			<ul class="decisions">
				{#each preview.decisions as decision (decision.scope)}
					<li class:granted={decision.granted}>
						<Icon icon={decision.granted ? RiCheckLine : RiCloseLine} size="0.9375rem" />
						<code>{decision.scope}</code>
						<span class="kind">
							{decision.is_default ? 'Default' : decision.kind === 'api' ? 'API' : 'OpenID'}
						</span>
						<span class="reason">{decision.reason}</span>
					</li>
				{/each}
			</ul>
		{/if}

		{#each blocks(preview) as block (block.title)}
			<CodeBlock title={block.title} code={json(block.value)} />
		{/each}
	</div>
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

	.muted {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.segments {
		display: inline-flex;
		align-self: flex-start;
		gap: 2px;
		padding: 3px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary);
	}

	.segments button {
		height: 30px;
		padding: 0 var(--space-3);
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text-hint);
		font: inherit;
		font-size: var(--text-sm);
		cursor: pointer;
	}

	.segments button.on {
		background: var(--color-surface);
		color: var(--color-text);
		font-weight: 600;
	}

	.segments button:disabled {
		cursor: not-allowed;
		opacity: 0.5;
	}

	.picked {
		display: inline-flex;
		align-items: center;
		align-self: flex-start;
		gap: var(--space-2);
		padding: 2px 2px 2px var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		font-size: var(--text-sm);
	}

	.user-search {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
	}

	.search {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		height: 40px;
		padding: 0 12px;
		border-bottom: 1px solid var(--color-border);
		background: var(--color-input);
		color: var(--color-text-hint);
	}

	.search input {
		flex: 1;
		min-width: 0;
		border: none;
		background: transparent;
		color: var(--color-text);
		font: inherit;
	}

	.search input:focus {
		outline: none;
	}

	.results {
		margin: 0;
		padding: var(--space-1);
		list-style: none;
	}

	.results li.muted {
		padding: var(--space-2);
	}

	.results button {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2);
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text);
		font: inherit;
		font-size: var(--text-sm);
		text-align: left;
		cursor: pointer;
	}

	.results button:hover {
		background: var(--row-hover);
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 6px;
	}

	.chips button {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		height: 28px;
		padding: 0 10px;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		background: transparent;
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		cursor: pointer;
	}

	.chips button.on {
		border-color: var(--color-text);
		background: var(--color-text);
		color: var(--color-surface);
	}

	.extra {
		height: 36px;
		padding: 0 12px;
		border: 1px solid var(--color-input-border);
		border-radius: var(--radius-sm);
		background: var(--color-input);
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.extra:focus {
		border-color: var(--color-brand);
		outline: none;
		box-shadow: inset 0 0 0 1px var(--color-brand);
	}

	.param {
		overflow-wrap: anywhere;
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-xs);
	}

	.run {
		display: flex;
		justify-content: flex-end;
	}

	.result {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		margin-top: var(--space-4);
	}

	.verdict {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		padding: var(--space-3);
		border-radius: var(--radius-md);
		background: var(--surface-danger);
		color: var(--color-danger);
	}

	.verdict.issued {
		background: var(--surface-success);
		color: var(--color-success);
	}

	.verdict strong {
		color: var(--color-text);
	}

	.verdict p {
		margin: 2px 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.decisions {
		margin: 0;
		padding: 0;
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		list-style: none;
	}

	.decisions li {
		display: grid;
		grid-template-columns: auto minmax(8rem, auto) auto 1fr;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		color: var(--color-danger);
	}

	.decisions li.granted {
		color: var(--color-success);
	}

	.decisions li + li {
		border-top: 1px solid var(--color-border);
	}

	.decisions code {
		color: var(--color-text);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.kind {
		padding: 1px 6px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.reason {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	@media (max-width: 34rem) {
		.decisions li {
			grid-template-columns: auto 1fr auto;
		}

		.reason {
			grid-column: 2 / -1;
		}
	}
</style>
