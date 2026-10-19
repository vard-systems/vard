<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { RiSettings3Line, RiTranslate2 } from '#lib/icons.js';
	import { languagesApi, type Language, type LocaleApp } from '#lib/api/index.js';
	import {
		Button,
		ConfirmDialog,
		DangerZone,
		FullscreenDialog,
		FieldGrid,
		FormSection,
		Input,
		Note,
		SwitchField,
		Tabs,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import TranslationEditor from './TranslationEditor.svelte';

	type Props = {
		open: boolean;
		language?: Language | null;
		canWrite: boolean;
		tab?: 'settings' | LocaleApp;
	};

	let {
		open = $bindable(false),
		language = null,
		canWrite,
		tab: startingTab = 'settings'
	}: Props = $props();

	const queryClient = useQueryClient();

	let tab = $state<string>('settings');
	let name = $state('');
	let native = $state('');
	let enabled = $state(false);
	let isDefault = $state(false);
	let position = $state('');

	let drafts = $state<Record<string, Record<string, string> | null>>({});
	let dirty = $state<Record<string, boolean>>({});

	let saving = $state(false);
	let confirmingDelete = $state(false);

	const apps = $derived(language?.apps ?? []);

	$effect(() => {
		if (!open || !language) return;

		tab = startingTab;
		confirmingDelete = false;
		name = language.name;
		native = language.native_name;
		enabled = language.is_enabled;
		isDefault = language.is_default;
		position = String(language.position);
		drafts = Object.fromEntries(apps.map((app) => [app, null]));
		dirty = {};
	});

	const locked = $derived(language?.base === true || isDefault);

	const settingsChanged = $derived(
		language !== null &&
			(name.trim() !== language.name ||
				native.trim() !== language.native_name ||
				enabled !== language.is_enabled ||
				isDefault !== language.is_default ||
				Number(position) !== language.position)
	);

	const textChanged = $derived(apps.filter((app) => dirty[app]));

	const ready = $derived(
		name.trim() !== '' && native.trim() !== '' && Number.isInteger(Number(position))
	);

	function appName(app: LocaleApp): string {
		return app === 'id' ? 'Sign-in pages' : app;
	}

	const tabs = $derived([
		{ value: 'settings', label: 'Settings', icon: RiSettings3Line },
		...apps.map((app) => ({
			value: app,
			label: appName(app),
			icon: RiTranslate2,
			count: (language?.missing[app] ?? 0) > 0 ? language?.missing[app] : undefined
		}))
	]);

	const save = createMutation(() => ({
		mutationFn: async () => {
			const code = language!.code;

			if (settingsChanged) {
				await languagesApi.update(code, {
					name: name.trim(),
					native_name: native.trim(),
					is_enabled: enabled,
					is_default: isDefault,
					position: Number(position)
				});
			}

			for (const app of textChanged) {
				await languagesApi.saveTranslation(code, app, drafts[app] ?? {});
			}

			return code;
		},
		onSuccess: async () => {
			notify.success(`${language?.name ?? 'Language'} saved`);
			open = false;

			// The editor seeds itself from its cached text, so what was cached
			// is dropped rather than refetched behind it: the next one opened
			// starts from what is saved now. English is every editor's left
			// column, which is why it is all of them and not only these.
			queryClient.removeQueries({ queryKey: keys.languages.translations });

			await queryClient.invalidateQueries({ queryKey: keys.languages.list });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this language');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	const remove = createMutation(() => ({
		mutationFn: () => languagesApi.remove(language?.code ?? ''),
		onSuccess: async () => {
			notify.success(`${language?.name ?? 'Language'} removed`);
			open = false;
			queryClient.removeQueries({ queryKey: keys.languages.translations });

			await queryClient.invalidateQueries({ queryKey: keys.languages.list });
		},
		onError: (err: unknown) => {
			confirmingDelete = false;
			notify.error(err, 'Could not remove this language');
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (!ready || saving || (!settingsChanged && textChanged.length === 0)) return;

		saving = true;
		save.mutate();
	}

	function defaultChanged(on: boolean) {
		if (on) enabled = true;
	}
</script>

{#snippet status()}
	{`Unsaved text: ${textChanged.map(appName).join(', ')}`}
{/snippet}

<FullscreenDialog
	bind:open
	status={textChanged.length > 0 ? status : undefined}
	title={language?.native_name ?? ''}
	description="Its names, where it is offered, and its text."
	meta={language?.code}
	onsubmit={submit}
>
	{#if language}
		<Tabs {tabs} bind:value={tab} label="Languages">
			{#snippet panel(value)}
				<div class="panel">
					{#if value === 'settings'}
						<FormSection title="Names" description="What it is called, here and to its readers.">
							<FieldGrid>
								<Input
									label="Name in English"
									bind:value={name}
									hint="What this panel calls it."
									readOnly={!canWrite}
									required
								/>
								<Input
									label="Name in itself"
									bind:value={native}
									hint="What the language picker shows, so people can find their own."
									readOnly={!canWrite}
									lang={language.code}
									required
								/>
							</FieldGrid>
						</FormSection>

						<FormSection title="Translated" description="How much of each app's text it has.">
							<div class="coverage">
								{#each apps as app (app)}
									{@const percent = language.coverage[app] ?? 0}
									<div class="row">
										<span class="app">{appName(app)}</span>
										<span class="track">
											<span class="fill" class:whole={percent === 100} style="--filled: {percent}%"
											></span>
										</span>
										<span class="value">
											{percent}%
											{#if (language.missing[app] ?? 0) > 0}
												<small>· {`${language.missing[app] ?? 0} missing`}</small>
											{/if}
										</span>
									</div>
								{/each}
							</div>

							<Note>
								{#if language.base}
									This is the language every other is a translation of: a key another language has
									no text for is shown in it. It cannot be turned off or removed.
								{:else if language.shipped}
									The server ships a translation of this language, so a key a new release adds is
									filled in on the next start. Nothing written here is overwritten.
								{:else}
									Added on this page. A key it has no text for is shown in English until somebody
									translates it.
								{/if}
							</Note>
						</FormSection>

						<FormSection
							title="Where it is offered"
							description="Whether people can pick it, and where it sits in the list."
						>
							<SwitchField
								label="The default language"
								description="What somebody sees before they have chosen. Making this the default takes the mark from whichever language has it."
								bind:checked={isDefault}
								onChange={defaultChanged}
								disabled={!canWrite || language.is_default}
							/>

							<SwitchField
								label="Offer this language"
								description={locked
									? 'The default language is always offered.'
									: 'Show it in the language picker on the sign-in pages.'}
								bind:checked={enabled}
								disabled={!canWrite || locked}
							/>

							<div class="position">
								<Input
									label="Place in the picker"
									bind:value={position}
									hint="Lower comes first."
									type="number"
									min="0"
									step="1"
									readOnly={!canWrite}
								/>
							</div>
						</FormSection>

						{#if canWrite && !language.base && !language.is_default}
							<DangerZone
								title="Remove this language"
								description="Its text goes with it, and anybody who chose it gets the default from their next page."
								onclick={() => (confirmingDelete = true)}
							/>
						{/if}
					{:else}
						{@const app = value as LocaleApp}
						<TranslationEditor
							code={language.code}
							name={name || language.name}
							native={native || language.native_name}
							{app}
							readOnly={!canWrite}
							bind:draft={drafts[app]}
							onDirty={(changed) => (dirty[app] = changed)}
						/>
					{/if}
				</div>
			{/snippet}
		</Tabs>
	{/if}

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)}>Cancel</Button>
		{#if canWrite}
			<Button
				type="submit"
				loading={saving}
				disabled={!ready || saving || (!settingsChanged && textChanged.length === 0)}
			>
				Save changes
			</Button>
		{/if}
	{/snippet}
</FullscreenDialog>

<ConfirmDialog
	bind:open={confirmingDelete}
	title={`Remove ${language?.name ?? 'this language'}?`}
	description="Its text goes with it, and anybody who chose it gets the default from their next page."
	confirmLabel="Remove language"
	busy={remove.isPending}
	onConfirm={() => remove.mutate()}
/>

<style>
	.panel {
		padding-top: var(--space-4);
	}

	.coverage {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.row {
		display: grid;
		grid-template-columns: 10rem 1fr auto;
		align-items: center;
		gap: var(--space-3);
	}

	.app {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.track {
		height: 8px;
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
	}

	.fill.whole {
		background: var(--color-success);
	}

	.value {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}

	.position {
		max-width: 16rem;
	}

	@media (max-width: 36rem) {
		.row {
			grid-template-columns: 1fr;
			gap: var(--space-1);
		}
	}
</style>
