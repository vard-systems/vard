<script lang="ts">
	import type { APIIntrospector } from '#lib/api/index.js';
	import { Note, Switch } from '#lib/components/ui/index.js';
	import { types } from '#lib/components/applications/applications.js';

	type Props = {
		introspectors: APIIntrospector[];
		editable: boolean;
		/** The application being saved, whose switch and every other wait for it. */
		saving: string | null;
		onChange: (app: APIIntrospector, listed: boolean) => void;
	};

	let { introspectors, editable, saving, onChange }: Props = $props();

	/** Only an application that proves who it is may be listed: the answer
	    carries the token's user, scope and roles. */
	const confidential = $derived(introspectors.filter((app) => app.confidential));
	const publicCount = $derived(introspectors.length - confidential.length);
</script>

{#if confidential.length === 0}
	<Note>
		No confidential application to list. Register the API's own backend as a web or
		machine-to-machine application, then allow it here.
	</Note>
{:else}
	<ul class="list">
		{#each confidential as app (app.id)}
			{@const kind = types[app.type]}
			<li class:listed={app.listed} class:busy={saving === app.id}>
				<span class="kind" title={kind.label}><i class={kind.icon} aria-hidden="true"></i></span>
				<div class="identity">
					<strong>{app.name}</strong>
					<code>{app.client_id}</code>
				</div>
				<Switch
					label={app.listed ? 'May introspect' : 'May not introspect'}
					checked={app.listed}
					disabled={!editable || saving !== null}
					onChange={(on) => onChange(app, on)}
				/>
			</li>
		{/each}
	</ul>
{/if}

{#if publicCount > 0}
	<p class="quiet">
		{publicCount}
		{publicCount === 1 ? 'public client is' : 'public clients are'} not shown: one that cannot prove who
		it is may never introspect.
	</p>
{/if}

<style>
	.list {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	/* As the API's applications above are drawn, so the two lists read as one page. */
	li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		transition: opacity var(--speed-fast);
	}

	li.listed {
		border-color: color-mix(in srgb, var(--color-success), var(--color-border) 55%);
	}

	li.busy {
		opacity: 0.6;
	}

	.kind {
		display: grid;
		flex-shrink: 0;
		place-items: center;
		width: 32px;
		height: 32px;
		border-radius: var(--radius-sm);
		background: var(--color-secondary-alt);
		color: var(--color-text-hint);
	}

	.identity {
		display: flex;
		flex: 1;
		flex-direction: column;
		gap: 1px;
		min-width: 0;
	}

	.identity code {
		overflow-wrap: anywhere;
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-xs);
	}

	.quiet {
		margin: var(--space-2) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
