<script lang="ts">
	import { enhance } from '$app/forms';
	import { useQueryClient } from '@tanstack/svelte-query';
	import {
		type MailContent,
		type MailLanguageContent,
		type MailMessageSpec
	} from '#lib/api/index.js';
	import {
		Button,
		Code,
		FormSection,
		Input,
		SaveBar,
		Select,
		Tag,
		Textarea,
		notify
	} from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { submit } from '#lib/forms.js';

	type Props = {
		content: MailContent;
	};

	let { content }: Props = $props();

	const queryClient = useQueryClient();

	// svelte-ignore state_referenced_locally
	let code = $state(content.languages[0]?.code ?? 'en');

	const language = $derived(
		content.languages.find((one) => one.code === code) ?? content.languages[0]
	);

	const choices = $derived(
		content.languages.map((one) => ({
			value: one.code,
			label: `${one.name} (${one.code})`,
			description: one.offered ? undefined : 'Not offered on the sign-in pages'
		}))
	);

	function wordsOf(one: MailLanguageContent | undefined): Record<string, string> {
		const words: Record<string, string> = {};

		for (const spec of content.messages) {
			words[spec.subject_key] = one?.messages[spec.subject_key] ?? '';
			words[spec.body_key] = one?.messages[spec.body_key] ?? '';
		}

		return words;
	}

	/** What is being typed, by key. Filled in before the first render rather
	    than in an effect: a field cannot bind to a key that is not there yet. */
	// svelte-ignore state_referenced_locally
	let drafts = $state(wordsOf(language));

	// svelte-ignore state_referenced_locally
	let filledFrom = $state(code);

	$effect(() => {
		if (code === filledFrom) return;

		filledFrom = code;
		drafts = wordsOf(content.languages.find((one) => one.code === code));
	});

	const dirty = $derived(
		content.messages.some(
			(spec) =>
				drafts[spec.subject_key] !== (language?.messages[spec.subject_key] ?? '') ||
				drafts[spec.body_key] !== (language?.messages[spec.body_key] ?? '')
		)
	);

	let saving = $state(false);

	const save = submit<unknown>({
		input: () => [code, drafts],
		when: () => !saving && dirty,
		pending: (busy) => (saving = busy),
		done: async () => {
			notify.success('Saved', 'The next message in this language uses these words.');
			await Promise.all([
				queryClient.invalidateQueries({ queryKey: keys.mail.content }),
				queryClient.invalidateQueries({ queryKey: keys.languages.all }),
				queryClient.invalidateQueries({ queryKey: keys.admin.overview })
			]);
		},
		failed: (err) => notify.error(err, 'Could not save these words')
	});

	function revert(spec: MailMessageSpec) {
		drafts = { ...drafts, [spec.subject_key]: '', [spec.body_key]: '' };
	}

	function shipped(key: string): string {
		return language?.base[key] ?? '';
	}

	function written(spec: MailMessageSpec): boolean {
		return drafts[spec.subject_key] !== '' || drafts[spec.body_key] !== '';
	}
</script>

<form method="POST" action="/actions/mail?/saveContent" use:enhance={save}>
	<FormSection
		title="Language"
		description="An email goes out in the reader's language. A field left empty falls back to the English this release ships, which is what its placeholder shows."
		icon="ri-translate-2"
	>
		{#snippet meta()}
			{#if language && !language.offered}
				<Tag tone="warning" small>Not offered</Tag>
			{/if}
		{/snippet}

		<Select
			label="Writing in"
			bind:value={code}
			options={choices}
			hint="The same text the Languages page holds. Add and remove languages there."
		/>
	</FormSection>

	{#each content.messages as spec (spec.kind)}
		<FormSection title={spec.label} description={spec.description} icon="ri-file-text-line">
			{#snippet meta()}
				{#if written(spec)}
					<Tag tone="info" small>Written</Tag>
				{:else}
					<Tag small>Shipped text</Tag>
				{/if}
			{/snippet}

			{#snippet action()}
				{#if written(spec)}
					<Button size="sm" variant="subtle" icon="ri-restart-line" onclick={() => revert(spec)}>
						Use the shipped text
					</Button>
				{/if}
			{/snippet}

			<Input
				label="Subject"
				bind:value={drafts[spec.subject_key]}
				maxlength={200}
				placeholder={shipped(spec.subject_key)}
			/>

			<Textarea
				label="Body"
				bind:value={drafts[spec.body_key]}
				rows={8}
				placeholder={shipped(spec.body_key)}
			/>

			<p class="params">
				{#each spec.params as param (param.name)}
					<Code tone="quiet" title={param.description}>{'{' + param.name + '}'}</Code>
				{/each}
				<span>are filled in when the message is sent.</span>
			</p>
		</FormSection>
	{/each}

	{#if dirty}
		<SaveBar {saving} ondiscard={() => (drafts = wordsOf(language))} />
	{/if}
</form>

<style>
	form > :global(.save-bar) {
		margin-top: var(--space-5);
	}

	.params {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-1) var(--space-2);
		margin: 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
