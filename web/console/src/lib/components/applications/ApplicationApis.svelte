<script lang="ts">
	import { resolve } from '$app/paths';
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { RiCodeBoxLine, RiShieldCheckLine } from 'svelte-remixicon';
	import { applicationsApi, type APIAccess } from '$lib/api';
	import { Alert, Badge, Checkbox, Icon, Note, Switch, notify } from '$lib/components/ui';
	import { keys, applicationAccessOptions } from '$lib/query';

	type Props = {
		applicationId: string;
		/** Whether the administrator may change what the application may do. */
		editable: boolean;
	};

	let { applicationId, editable }: Props = $props();

	const queryClient = useQueryClient();

	const access = createQuery(() => applicationAccessOptions(applicationId));

	/** The API being saved, so only its card shows the wait. */
	let saving = $state<string | null>(null);

	/** Changes are saved as they are made, the way the role mapping is: the
	    answer is the whole list, so the cache is set straight from it. */
	const change = createMutation(() => ({
		mutationFn: ({ api, scopes }: { api: APIAccess; scopes: string[] | null }) =>
			scopes === null
				? applicationsApi.revokeAPI(applicationId, api.id)
				: applicationsApi.authorizeAPI(applicationId, api.id, scopes),
		onSuccess: async (result: { apis: APIAccess[] }, { api, scopes }) => {
			notify.success(
				scopes === null
					? `Access to ${api.name} removed`
					: api.authorized
						? `Access to ${api.name} saved`
						: `Allowed to call ${api.name}`
			);
			queryClient.setQueryData(keys.applications.access(applicationId), result.apis);
			await queryClient.invalidateQueries({ queryKey: keys.apis.all });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not change this API access');
		},
		onSettled: () => {
			saving = null;
		}
	}));

	function save(api: APIAccess, scopes: string[] | null) {
		if (saving) return;

		saving = api.id;
		change.mutate({ api, scopes });
	}

	function allowed(api: APIAccess): string[] {
		return api.scopes.filter((scope) => scope.allowed).map((scope) => scope.id);
	}

	function toggleScope(api: APIAccess, id: string, on: boolean) {
		const now = allowed(api);
		save(api, on ? [...now, id] : now.filter((it) => it !== id));
	}

	function setAll(api: APIAccess, on: boolean) {
		save(api, on ? api.scopes.map((scope) => scope.id) : []);
	}

	const authorizedCount = $derived((access.data ?? []).filter((api) => api.authorized).length);
</script>

<p class="lead">
	Which APIs this application may ask for tokens for, and the most each token may carry. A user's
	token gets an allowed scope only when their roles grant it too, if the API enforces roles.
</p>

{#if access.isPending}
	<Note>Loading APIs…</Note>
{:else if access.isError}
	<Alert>Could not load the APIs.</Alert>
{:else if access.data.length === 0}
	<div class="empty">
		<Icon icon={RiCodeBoxLine} size="1.25rem" />
		<div>
			<strong>No APIs yet</strong>
			<p>
				Register an API on the
				<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
				<a href={resolve('/admin/(panel)/dashboard/apis')}>APIs page</a>, then allow this
				application to use it here.
			</p>
		</div>
	</div>
{:else}
	<p class="summary">
		Authorised for <strong>{authorizedCount}</strong> of {access.data.length}
		{access.data.length === 1 ? 'API' : 'APIs'}
	</p>

	<div class="apis">
		{#each access.data as api (api.id)}
			{@const count = allowed(api).length}
			<section class:authorized={api.authorized} class:busy={saving === api.id}>
				<header>
					<div class="identity">
						<strong>{api.name}</strong>
						<code>{api.identifier}</code>
					</div>

					{#if api.enforce_roles}
						<span class="enforced" title="A user's token gets only the scopes their roles grant">
							<Icon icon={RiShieldCheckLine} size="0.875rem" />
							Role-based
						</span>
					{/if}

					<Switch
						label={api.authorized ? 'Authorised' : 'Not authorised'}
						checked={api.authorized}
						disabled={!editable || saving !== null}
						onChange={(on) => save(api, on ? [] : null)}
					/>
				</header>

				{#if api.authorized}
					<div class="scopes">
						<div class="scopes-head">
							<span>
								{count} of {api.scopes.length}
								{api.scopes.length === 1 ? 'scope' : 'scopes'} allowed
							</span>
							{#if editable && api.scopes.length > 1}
								<button
									type="button"
									onclick={() => setAll(api, count < api.scopes.length)}
									disabled={saving !== null}
								>
									{count < api.scopes.length ? 'Allow all' : 'Allow none'}
								</button>
							{/if}
						</div>

						{#if api.scopes.length === 0}
							<Note>
								This API has no scopes: tokens for it carry its audience and nothing else.
							</Note>
						{:else}
							<ul>
								{#each api.scopes as scope (scope.id)}
									<li>
										<Checkbox
											checked={scope.allowed}
											onChange={(on) => toggleScope(api, scope.id, on)}
											disabled={!editable || saving !== null}
											title={scope.name}
										/>
										<div>
											<code>{scope.name}</code>
											{#if scope.description}
												<span class="muted">{scope.description}</span>
											{/if}
										</div>
									</li>
								{/each}
							</ul>
						{/if}
					</div>
				{:else if api.scopes.length > 0}
					<p class="off">
						{api.scopes.length}
						{api.scopes.length === 1 ? 'scope' : 'scopes'}:
						{#each api.scopes.slice(0, 4) as scope (scope.id)}
							<Badge>{scope.name}</Badge>
						{/each}
						{#if api.scopes.length > 4}
							<span class="muted">+{api.scopes.length - 4}</span>
						{/if}
					</p>
				{/if}
			</section>
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

	.muted {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.summary {
		margin: 0 0 var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.summary strong {
		color: var(--color-text);
	}

	.apis {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	section {
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		transition: opacity var(--speed-fast);
	}

	section.authorized {
		border-color: color-mix(in srgb, var(--color-success), var(--color-border) 55%);
	}

	section.busy {
		opacity: 0.6;
	}

	header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2) var(--space-3);
		padding: var(--space-3);
	}

	.identity {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 12rem;
	}

	code {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}

	.identity code {
		color: var(--color-text-hint);
	}

	.enforced {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.scopes {
		padding: var(--space-2) var(--space-3) var(--space-3);
		border-top: 1px solid var(--color-border);
	}

	.scopes-head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-bottom: var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.scopes-head button {
		padding: 0;
		border: none;
		background: none;
		color: var(--color-text);
		font: inherit;
		font-weight: 600;
		cursor: pointer;
	}

	ul {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(15rem, 1fr));
		gap: var(--space-2) var(--space-4);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
	}

	li > div {
		display: flex;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
	}

	.off {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-1);
		margin: 0;
		padding: 0 var(--space-3) var(--space-3);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.empty {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-4);
		border: 1px dashed var(--color-border);
		border-radius: var(--radius-surface);
		color: var(--color-text-hint);
	}

	.empty strong {
		color: var(--color-text);
	}

	.empty p {
		margin: 2px 0 0;
		font-size: var(--text-sm);
	}

	.empty a {
		color: var(--color-text);
	}
</style>
