<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import type { LocaleApp } from '#lib/api/index.js';
	import { countOf } from '#lib/utils/format.js';
	import { Alert, Button, SearchInput, Switch, notify } from '#lib/components/ui/index.js';
	import { nest } from './flatten';
	import { translationOptions } from '#lib/query/index.js';
	import { download, missingParameters, readTranslationFile } from './translations';

	type Props = {
		code: string;
		name: string;
		native: string;
		app: LocaleApp;
		readOnly?: boolean;
		draft: Record<string, string> | null;
		onDirty: (dirty: boolean) => void;
	};

	let {
		code,
		name,
		native,
		app,
		readOnly = false,
		draft = $bindable(null),
		onDirty
	}: Props = $props();

	const saved = createQuery(() => translationOptions(code, app));

	$effect(() => {
		if (draft === null && saved.data) draft = { ...saved.data.messages };
	});

	$effect(() => {
		const original = saved.data?.messages ?? {};
		const current = draft ?? original;

		const keys = new Set([...Object.keys(original), ...Object.keys(current)]);
		onDirty([...keys].some((key) => (original[key] ?? '').trim() !== (current[key] ?? '').trim()));
	});

	let search = $state('');
	let onlyMissing = $state(false);

	const translated = (key: string) => (draft?.[key] ?? '').trim() !== '';

	const total = $derived(saved.data?.keys.length ?? 0);
	const done = $derived(saved.data?.keys.filter(translated).length ?? 0);

	const visible = $derived.by(() => {
		if (!saved.data) return [];

		const term = search.trim().toLowerCase();
		const base = saved.data.base;

		return saved.data.keys.filter((key) => {
			if (onlyMissing && translated(key)) return false;
			if (term === '') return true;

			return [key, base[key] ?? '', draft?.[key] ?? ''].some((value) =>
				value.toLowerCase().includes(term)
			);
		});
	});

	function edit(key: string, value: string) {
		draft = { ...draft, [key]: value };
	}

	let fileInput = $state<HTMLInputElement>();

	async function importFile(event: Event) {
		const input = event.currentTarget as HTMLInputElement;
		const file = input.files?.[0];
		input.value = '';

		if (!file || !saved.data) return;

		const result = await readTranslationFile(file, saved.data.keys);
		if (!result.ok) {
			notify.problem(
				`${file.name} is not a translation file: it has to be one JSON object of texts by key.`
			);
			return;
		}

		draft = { ...draft, ...result.messages };

		notify.success(
			`${countOf(Object.keys(result.messages).length, 'text')} imported from ${file.name}`,
			result.skipped > 0
				? `Save to keep them. Skipped ${countOf(result.skipped, 'key')} this version does not use.`
				: 'Save to keep them.'
		);
	}

	function exportFile() {
		if (!saved.data) return;

		const body: Record<string, string> = { $name: name, $native: native };
		for (const key of saved.data.keys) body[key] = draft?.[key] ?? '';

		download(`${code}.${app}.json`, JSON.stringify(nest(body), null, 2) + '\n');
	}
</script>

<div class="editor">
	{#if saved.isPending}
		<p class="state">Loading the text…</p>
	{:else if saved.isError || !saved.data}
		<Alert>Could not load the text. Close this and try again.</Alert>
	{:else}
		<div class="toolbar">
			<SearchInput label="Search" placeholder="Search keys and text…" bind:value={search} />

			<Switch label="Only untranslated" bind:checked={onlyMissing} />
		</div>

		<div class="summary">
			<span class="progress">
				<span class="track">
					<span class="fill" class:whole={done === total} style="--filled: {(done * 100) / total}%"
					></span>
				</span>
				{`${done} of ${total} translated`}
			</span>

			<span class="files">
				{#if !readOnly}
					<input
						id={`translation-${code}-${app}-file`}
						name={`translation-${code}-${app}-file`}
						bind:this={fileInput}
						class="file"
						type="file"
						accept=".json,application/json"
						onchange={importFile}
						tabindex="-1"
						aria-hidden="true"
					/>
					<Button size="sm" variant="subtle" onclick={() => fileInput?.click()}>
						<i class="ri-upload-2-line" aria-hidden="true"></i>
						Import JSON
					</Button>
				{/if}
				<Button size="sm" variant="subtle" onclick={exportFile}>
					<i class="ri-download-2-line" aria-hidden="true"></i>
					Export JSON
				</Button>
			</span>
		</div>

		{#if visible.length === 0}
			<p class="state">Nothing matches.</p>
		{:else}
			<ol class="keys">
				{#each visible as key (key)}
					{@const english = saved.data.base[key] ?? ''}
					{@const value = draft?.[key] ?? ''}
					{@const lost = value.trim() ? missingParameters(english, value) : []}
					<li class:missing={!value.trim()}>
						<div class="source">
							<code>{key}</code>
							<p lang="en">{english}</p>
						</div>

						<div class="target">
							<textarea
								id={`translation-${code}-${app}-${key}`}
								name={`translation-${code}-${app}-${key}`}
								{value}
								lang={code}
								rows="1"
								placeholder={english}
								readonly={readOnly}
								aria-label={key}
								oninput={(event) => edit(key, event.currentTarget.value)}></textarea>

							{#if lost.length > 0}
								<small class="warning">
									<i class="ri-error-warning-line" aria-hidden="true"></i>
									{`Keep ${lost.join(', ')} in the text: it is filled in when the page is shown.`}
								</small>
							{/if}
						</div>
					</li>
				{/each}
			</ol>
		{/if}
	{/if}
</div>

<style>
	.editor {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.state {
		margin: var(--space-4) 0;
		color: var(--color-text-hint);
		text-align: center;
	}

	.toolbar {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.summary {
		display: flex;
		align-items: center;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: var(--space-2);
	}

	.progress {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-variant-numeric: tabular-nums;
	}

	.track {
		display: block;
		width: 6rem;
		height: 6px;
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		overflow: hidden;
	}

	.fill {
		display: block;
		width: var(--filled);
		height: 100%;
		border-radius: inherit;
		background: var(--color-text-hint);
		transition: width var(--speed-fast);
	}

	.fill.whole {
		background: var(--color-success);
	}

	.files {
		display: inline-flex;
		gap: var(--space-2);
	}

	.file {
		display: none;
	}

	.keys {
		display: flex;
		flex-direction: column;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		display: grid;
		grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr);
		align-items: start;
		gap: var(--space-3);
		padding: var(--space-3) 0;
		border-top: 1px solid var(--color-border);
	}

	.source {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	code {
		color: var(--color-text-hint);
		font-family: var(--font-mono);
		font-size: var(--text-xs);
		overflow-wrap: anywhere;
	}

	.source p {
		margin: 0;
		line-height: 1.45;
		overflow-wrap: anywhere;
	}

	.target {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	textarea {
		width: 100%;
		min-height: var(--control-height);
		padding: 10px 13px;
		border: 1px solid var(--color-input-border);
		border-radius: var(--radius-field);
		background: var(--color-input);
		color: var(--color-text);
		font: inherit;
		font-size: var(--text-base);
		line-height: 1.45;
		resize: vertical;
		field-sizing: content;
		transition:
			border-color var(--speed-fast),
			box-shadow var(--speed-fast);
	}

	textarea:focus {
		border-color: var(--color-brand);
		box-shadow: inset 0 0 0 1px var(--color-brand);
		outline: none;
	}

	textarea::placeholder {
		color: var(--color-text-hint);
		opacity: 0.7;
	}

	li.missing textarea {
		border-style: dashed;
	}

	li.missing textarea:focus {
		border-style: solid;
	}

	.warning {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--color-warning);
		font-size: var(--text-sm);
	}

	@media (max-width: 40rem) {
		li {
			grid-template-columns: 1fr;
			gap: var(--space-2);
		}
	}
</style>
