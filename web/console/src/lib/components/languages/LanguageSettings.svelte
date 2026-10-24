<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import type { Language, LanguageList, LocaleApp } from '#lib/api/index.js';
	import { Button, SearchInput, ShowMore, Toolbar } from '#lib/components/ui/index.js';
	import { firstPage, languagesOptions, LIST_PAGE_SIZE } from '#lib/query/index.js';
	import LanguageDialog from './LanguageDialog.svelte';
	import LanguageTable from './LanguageTable.svelte';
	import NewLanguageDialog from './NewLanguageDialog.svelte';

	type Props = {
		initial: LanguageList;
		canWrite: boolean;
	};

	let { initial, canWrite }: Props = $props();

	const list = createQuery(() => languagesOptions(initial));

	let search = $state('');

	const visible = $derived.by(() => {
		const term = search.trim().toLowerCase();
		if (term === '') return list.data.languages;

		return list.data.languages.filter((language) =>
			[language.name, language.native_name, language.code].some((value) =>
				value.toLowerCase().includes(term)
			)
		);
	});

	let shown = $derived(firstPage(search));
	const rows = $derived(visible.slice(0, shown));

	let editing = $state<Language | null>(null);
	let dialogTab = $state<'settings' | LocaleApp>('settings');
	let dialogOpen = $state(false);
	let creating = $state(false);

	function open(language: Language, tab: 'settings' | LocaleApp = 'settings') {
		editing = language;
		dialogTab = tab;
		dialogOpen = true;
	}

	function created(language: Language) {
		const unfinished = language.apps.find((app) => (language.missing[app] ?? 0) > 0);
		open(language, unfinished ?? 'settings');
	}
</script>

<div class="languages">
	<p class="lead">
		The languages the sign-in pages can be shown in. Their text lives in the database: add a
		language, translate it here or import a file, and it is on the next page anybody opens — nothing
		is rebuilt. This panel is English and is not one of them.
	</p>

	<Toolbar>
		<SearchInput label="Search" placeholder="Search name or code…" bind:value={search} />

		{#snippet end()}
			<span class="total">{`${list.data.languages.length} available`}</span>

			{#if canWrite}
				<Button onclick={() => (creating = true)}>
					<i class="ri-add-line" aria-hidden="true"></i>
					New language
				</Button>
			{/if}
		{/snippet}
	</Toolbar>

	<LanguageTable
		languages={rows}
		apps={list.data.apps}
		onOpen={open}
		empty="No languages match this."
	/>

	{#if visible.length > shown}
		<ShowMore {shown} total={visible.length} onclick={() => (shown += LIST_PAGE_SIZE)} />
	{/if}
</div>

<LanguageDialog bind:open={dialogOpen} language={editing} tab={dialogTab} {canWrite} />

<NewLanguageDialog bind:open={creating} languages={list.data.languages} onCreated={created} />

<style>
	.languages {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.lead {
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		line-height: 1.5;
	}

	.total {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		white-space: nowrap;
	}
</style>
