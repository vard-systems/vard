<!--
	@component
	Searches every page and heading, drawn like the console's command palette. The index
	(routes/search.json) is built ahead and fetched on first open; / or ⌘K opens it.
-->
<script lang="ts">
	import { goto } from '$app/navigation';
	import { RiCornerDownLeftLine, RiFileList3Line, RiSearchLine } from 'svelte-remixicon';
	import { href } from '#lib/docs/links.js';
	import type { SearchEntry } from '#lib/docs/types.js';
	import Icon from '../Icon.svelte';

	let dialog = $state<HTMLDialogElement>();
	let input = $state<HTMLInputElement>();
	let query = $state('');
	let active = $state(0);
	/** An entry, with what search compares lower-cased once when it loads
	    rather than on every keystroke. */
	type Indexed = { entry: SearchEntry; title: string; method: string; haystack: string };
	let index = $state<Indexed[]>([]);
	let failed = $state(false);
	let loading: Promise<void> | undefined;

	const LIMIT = 40;

	function load() {
		loading ??= fetch(href('/search.json'))
			.then((response) => response.json())
			.then((entries: SearchEntry[]) => {
				index = entries.map((entry) => {
					const title = entry.title.toLowerCase();
					const method = (entry.method ?? '').toLowerCase();
					const haystack = `${method} ${title} ${entry.page} ${entry.group}`.toLowerCase();
					return { entry, title, method, haystack };
				});
			})
			.catch(() => {
				failed = true;
				loading = undefined;
			});
	}

	function show() {
		load();
		dialog?.showModal();
		input?.select();
	}

	function close() {
		dialog?.close();
	}

	// Every word has to appear; a match in the heading, and at its start,
	// ranks above one in the page it is on.
	const results = $derived.by(() => {
		const words = query.toLowerCase().split(/\s+/).filter(Boolean);
		if (words.length === 0) return [];

		const scored: { entry: SearchEntry; score: number }[] = [];
		for (const { entry, title, method, haystack } of index) {
			if (!words.every((word) => haystack.includes(word))) continue;

			let score = entry.href.includes('#') ? 0 : 1;
			for (const word of words) {
				if (title.startsWith(word)) score += 4;
				else if (title.includes(word)) score += 2;
				if (method === word) score += 3;
			}
			scored.push({ entry, score });
		}

		return scored
			.sort((a, b) => b.score - a.score)
			.slice(0, LIMIT)
			.map((s) => s.entry);
	});

	$effect(() => {
		void query;
		active = 0;
	});

	$effect(() => {
		dialog?.querySelector(`[data-index="${active}"]`)?.scrollIntoView({ block: 'nearest' });
	});

	function choose(entry: SearchEntry) {
		close();
		// eslint-disable-next-line svelte/no-navigation-without-resolve -- href() is resolve()
		goto(href(entry.href));
	}

	function onkeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			active = Math.min(active + 1, results.length - 1);
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			active = Math.max(active - 1, 0);
		} else if (event.key === 'Enter' && results[active]) {
			event.preventDefault();
			choose(results[active]);
		}
	}

	function onglobalkeydown(event: KeyboardEvent) {
		const typing =
			event.target instanceof HTMLElement && event.target.closest('input, textarea, select');
		if ((event.key === 'k' && (event.metaKey || event.ctrlKey)) || (event.key === '/' && !typing)) {
			event.preventDefault();
			show();
		}
	}
</script>

<svelte:window onkeydown={onglobalkeydown} />

<button
	type="button"
	class="control"
	data-tooltip="Search (press /)"
	aria-label="Search the docs"
	onclick={show}
	onfocus={load}
	onmouseenter={load}
>
	<Icon icon={RiSearchLine} />
</button>

<dialog
	bind:this={dialog}
	aria-label="Search"
	onclick={(event) => {
		if (event.target === dialog) close();
	}}
>
	<div class="field">
		<Icon icon={RiSearchLine} size="1.125rem" />
		<input
			bind:this={input}
			bind:value={query}
			{onkeydown}
			type="search"
			placeholder="Search guides, endpoints and errors"
			aria-label="Search"
			autocomplete="off"
			spellcheck="false"
		/>
	</div>

	<div class="results" role="listbox" aria-label="Results">
		{#each results as entry, i (entry.href)}
			<!-- eslint-disable svelte/no-navigation-without-resolve -- href() is resolve() -->
			<a
				href={href(entry.href)}
				class="row"
				role="option"
				aria-selected={i === active}
				data-index={i}
				data-active={i === active}
				onclick={(event) => {
					event.preventDefault();
					choose(entry);
				}}
				onmousemove={() => (active = i)}
			>
				<!-- eslint-enable svelte/no-navigation-without-resolve -->
				{#if entry.method}
					<span class="method method-{entry.method.toLowerCase()}">{entry.method}</span>
				{:else}
					<span class="glyph"><Icon icon={RiFileList3Line} /></span>
				{/if}
				<span class="label" class:path={entry.method}>{entry.title}</span>
				<span class="where">{entry.page}</span>
				{#if i === active}
					<Icon icon={RiCornerDownLeftLine} size="0.875rem" />
				{/if}
			</a>
		{/each}

		{#if failed}
			<p class="empty">The search index could not be loaded.</p>
		{:else if query && results.length === 0}
			<p class="empty">Nothing matches “{query}”.</p>
		{:else if !query}
			<p class="empty">Try “token”, “PKCE”, “POST users” or an error code.</p>
		{/if}
	</div>

	<footer>
		<span><kbd>↑</kbd><kbd>↓</kbd> move</span>
		<span><kbd>↵</kbd> open</span>
		<span><kbd>esc</kbd> close</span>
	</footer>
</dialog>

<style>
	/* The console's palette: below the top of the window, so the list grows
	   downwards without the sheet moving while being typed in. */
	dialog {
		width: min(36rem, calc(100vw - 2 * var(--space-4)));
		max-height: min(26rem, 70vh);
		margin: 12vh auto auto;
		padding: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-lg);
		background: var(--dialog-surface);
		color: var(--color-text);
		box-shadow: var(--shadow-md);
		overflow: hidden;
	}

	dialog[open] {
		display: flex;
		flex-direction: column;
		animation: palette-in var(--speed) ease;
	}

	dialog::backdrop {
		background: var(--color-overlay);
		backdrop-filter: blur(2px);
	}

	@keyframes palette-in {
		from {
			opacity: 0;
			transform: scale(0.98) translateY(-4px);
		}
	}

	.field {
		display: flex;
		flex-shrink: 0;
		align-items: center;
		gap: var(--space-2);
		height: var(--control-height-lg);
		padding: 0 var(--space-3);
		border-bottom: 1px solid var(--color-border);
		color: var(--color-text-hint);
	}

	.field input {
		flex: 1;
		min-width: 0;
		border: none;
		background: none;
		color: var(--color-text);
		font: inherit;
		font-size: var(--text-lg);
	}

	.field input:focus {
		outline: none;
	}

	.results {
		flex: 1;
		min-height: 0;
		padding: var(--space-1);
		overflow-y: auto;
	}

	.row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: 8px var(--space-2);
		border-radius: var(--radius-sm);
		color: var(--color-text);
		font-size: var(--text-base);
		text-decoration: none;
	}

	.row[data-active='true'] {
		background: var(--color-secondary);
	}

	.glyph {
		display: inline-flex;
		justify-content: center;
		min-width: 48px;
		color: var(--color-text-hint);
	}

	.label {
		flex: 1;
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.label.path {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
	}

	.where {
		flex: none;
		max-width: 40%;
		overflow: hidden;
		color: var(--color-text-hint);
		font-size: var(--text-xs);
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.empty {
		margin: 0;
		padding: var(--space-4);
		color: var(--color-text-hint);
		text-align: center;
	}

	footer {
		display: flex;
		flex-shrink: 0;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-3);
		border-top: 1px solid var(--color-border);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	footer span {
		display: inline-flex;
		align-items: center;
		gap: 3px;
	}

	@media (prefers-reduced-motion: reduce) {
		dialog[open] {
			animation: none;
		}
	}
</style>
