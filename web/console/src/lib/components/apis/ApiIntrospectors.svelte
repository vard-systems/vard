<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { apisApi, type API, type APIIntrospector } from '#lib/api/index.js';
	import { Alert, Note, notify } from '#lib/components/ui/index.js';
	import { apiIntrospectorsOptions, keys } from '#lib/query/index.js';
	import IntrospectorList from './IntrospectorList.svelte';

	type Props = { api: API; editable: boolean };

	let { api, editable }: Props = $props();

	const queryClient = useQueryClient();

	const introspectors = createQuery(() => apiIntrospectorsOptions(api.id));

	let saving = $state<string | null>(null);

	/** One switch is one change, saved as it is made: the whole list goes, with
	    this application in or out of it. */
	const change = createMutation(() => ({
		mutationFn: ({ app, listed }: { app: APIIntrospector; listed: boolean }) => {
			const others = (introspectors.data ?? [])
				.filter((it) => it.listed && it.id !== app.id)
				.map((it) => it.id);
			return apisApi.setIntrospectors(api.id, listed ? [...others, app.id] : others);
		},
		onSuccess: async (answer, { app, listed }) => {
			queryClient.setQueryData(keys.apis.introspectors(api.id), answer.introspectors);
			notify.success(
				listed ? `${app.name} may introspect this API's tokens` : `${app.name} no longer may`
			);
			await queryClient.invalidateQueries({ queryKey: keys.apis.logs(api.id) });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not change who may introspect');
		},
		onSettled: () => {
			saving = null;
		}
	}));

	function save(app: APIIntrospector, listed: boolean) {
		if (saving) return;

		saving = app.id;
		change.mutate({ app, listed });
	}
</script>

<section class="introspection" aria-labelledby="introspection-title">
	<h2 id="introspection-title">Token introspection</h2>
	<p class="lead">
		The API's own backend may ask this server whether an access token for this API is still live —
		revoked, its session ended, its user turned off — whichever application the token was issued to.
		The answer names the token's user, scope and roles, so list only the API's own clients. Every
		application may always introspect its own tokens.
	</p>

	{#if introspectors.isPending}
		<Note>Loading applications…</Note>
	{:else if introspectors.isError}
		<Alert>Could not load the applications.</Alert>
	{:else}
		<IntrospectorList introspectors={introspectors.data} {editable} {saving} onChange={save} />
	{/if}
</section>

<style>
	.introspection {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin-top: var(--space-6);
	}

	h2 {
		margin: 0;
		font-size: var(--text-lg);
	}

	.lead {
		margin: 0 0 var(--space-2);
		max-width: 75ch;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.5;
	}
</style>
