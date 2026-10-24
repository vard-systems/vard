<script lang="ts">
	import { untrack } from 'svelte';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { languagesApi, type Language } from '#lib/api/index.js';
	import {
		Button,
		FullscreenDialog,
		FieldGrid,
		FormSection,
		Input,
		Select,
		SwitchField,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { languageCode, namesOf } from './translations';

	type Props = {
		open: boolean;
		languages: Language[];
		onCreated: (language: Language) => void;
	};

	let { open = $bindable(false), languages, onCreated }: Props = $props();

	const queryClient = useQueryClient();

	let code = $state('');
	let name = $state('');
	let native = $state('');
	const NOTHING = '-';

	let copyFrom = $state(NOTHING);
	let enabled = $state(false);
	let isDefault = $state(false);

	let namesTyped = $state(false);

	let saving = $state(false);

	$effect(() => {
		if (!open) return;

		code = '';
		name = '';
		native = '';
		copyFrom = NOTHING;
		enabled = false;
		isDefault = false;
		namesTyped = false;
	});

	const trimmed = $derived(code.trim());

	const codeError = $derived.by(() => {
		if (trimmed === '') return undefined;
		if (!languageCode.test(trimmed))
			return 'A language tag is two or three letters, then optionally a region: de, pt-BR.';
		if (languages.some((one) => one.code.toLowerCase() === trimmed.toLowerCase())) {
			return 'There is already a language with this code.';
		}
		return undefined;
	});

	const ready = $derived(
		trimmed !== '' && codeError === undefined && name.trim() !== '' && native.trim() !== ''
	);

	$effect(() => {
		const current = trimmed;
		untrack(() => fillNames(current));
	});

	function fillNames(current: string) {
		if (namesTyped) return;

		const names = languageCode.test(current) ? namesOf(current) : null;
		name = names?.name ?? '';
		native = names?.native ?? '';
	}

	const sources = $derived([
		{ value: NOTHING, label: 'Nothing — untranslated text is shown in English' },
		...languages.map((language) => ({
			value: language.code,
			label: `A copy of ${language.native_name}`,
			description: language.name
		}))
	]);

	const create = createMutation(() => ({
		mutationFn: () =>
			languagesApi.create({
				code: trimmed,
				name: name.trim(),
				native_name: native.trim(),
				is_enabled: enabled || isDefault,
				is_default: isDefault,
				copy_from: copyFrom === NOTHING ? undefined : copyFrom
			}),
		onSuccess: async ({ language }) => {
			notify.success(`${language.name} added`);
			open = false;
			await queryClient.invalidateQueries({ queryKey: keys.languages.list });
			onCreated(language);
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not add this language');
		},
		onSettled: () => {
			saving = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (!ready || saving) return;

		saving = true;
		create.mutate();
	}
</script>

<FullscreenDialog
	bind:open
	title="New language"
	description="It starts off, so it can be translated before anybody is offered it."
	onsubmit={submit}
>
	<FormSection title="Names" description="The tag it is known by, and what it is called.">
		<Input
			label="Code"
			bind:value={code}
			hint="A language tag: uz, de, pt-BR."
			error={codeError}
			placeholder="uz"
			autocomplete="off"
			spellcheck={false}
			required
		/>

		<FieldGrid>
			<Input
				label="Name in English"
				bind:value={name}
				oninput={() => (namesTyped = true)}
				hint="What this panel calls it."
				required
			/>
			<Input
				label="Name in itself"
				bind:value={native}
				oninput={() => (namesTyped = true)}
				hint="What the language picker shows, so people can find their own."
				lang={trimmed || undefined}
				required
			/>
		</FieldGrid>
	</FormSection>

	<FormSection
		title="Text"
		description="Where its text starts from, and whether people can pick it yet."
	>
		<Select label="Start from" bind:value={copyFrom} options={sources} />

		<SwitchField
			label="Offer it now"
			description="Otherwise it stays off until you turn it on, so it can be translated first."
			bind:checked={enabled}
			disabled={isDefault}
		/>

		<SwitchField
			label="The default language"
			description="What the sign-in pages show somebody who has not chosen. It takes the mark from English, and is offered at once."
			bind:checked={isDefault}
		/>
	</FormSection>

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)}>Cancel</Button>
		<Button type="submit" loading={saving} disabled={!ready || saving}>Add language</Button>
	{/snippet}
</FullscreenDialog>
