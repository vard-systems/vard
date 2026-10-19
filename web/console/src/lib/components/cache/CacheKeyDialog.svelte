<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { RiDeleteBinLine, RiTimerLine } from '#lib/icons.js';
	import { cacheApi, messageOf, type CacheDatabaseName, type CacheKey } from '#lib/api/index.js';
	import {
		Alert,
		CodeEditor,
		DangerZone,
		Button,
		FullscreenDialog,
		FieldGrid,
		FormSection,
		Icon,
		Input,
		Note,
		Tag,
		notify
	} from '#lib/components/ui/index.js';
	import { keys, cacheValueOptions } from '#lib/query/index.js';
	import { formatBytes, formatTTL, kinds } from './cache';

	type Props = {
		database: CacheDatabaseName;
		key: CacheKey | null;
		open: boolean;
	};

	let { database, key, open = $bindable(false) }: Props = $props();

	const queryClient = useQueryClient();

	/** What the key holds is read when it is opened, not with the listing: a
	    page of keys would otherwise carry every value in it. */
	const value = createQuery(() => ({
		...cacheValueOptions(database, key?.name ?? ''),
		enabled: open && key !== null
	}));

	const current = $derived(value.data?.key);
	const editable = $derived(current?.editable ?? false);

	let ttl = $state('');
	let confirmingDelete = $state(false);

	$effect(() => {
		if (!open) return;
		confirmingDelete = false;
		ttl = '';
	});

	let text = $derived(current ? JSON.stringify(current.value, null, 2) : '');

	const parsed = $derived.by(() => {
		try {
			const value: unknown = JSON.parse(text);
			return value === null ? undefined : { value };
		} catch {
			return undefined;
		}
	});

	const ttlSeconds = $derived(ttl.trim() === '' ? undefined : Number(ttl));
	const ttlValid = $derived(
		ttlSeconds === undefined ||
			(Number.isInteger(ttlSeconds) && ttlSeconds > 0 && ttlSeconds <= 86400)
	);

	function refill() {
		return queryClient.invalidateQueries({ queryKey: keys.cache.all });
	}

	const save = createMutation(() => ({
		mutationFn: () =>
			cacheApi.update(database, key?.name ?? '', {
				value: parsed?.value,
				ttl_seconds: ttlSeconds
			}),
		onSuccess: async () => {
			notify.success('Value saved');
			await refill();
			open = false;
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this value');
		}
	}));

	const remove = createMutation(() => ({
		mutationFn: () => cacheApi.remove(database, key?.name ?? ''),
		onSuccess: async () => {
			notify.success('Key removed');
			await refill();
			open = false;
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not remove this key');
		}
	}));

	const busy = $derived(save.isPending || remove.isPending);

	function submit(event: SubmitEvent) {
		event.preventDefault();
		if (!editable || !parsed || !ttlValid || busy) return;
		save.mutate();
	}

	const consequence = $derived.by(() => {
		switch (current?.kind) {
			case 'session':
				return 'The next request with this session reads it from the database again. Nobody is signed out.';
			case 'ratelimit':
				return 'This address starts counting again from nothing.';
			case 'stale':
				return 'Nothing reads it any more; it would expire on its own.';
			default:
				return 'The next request that wants it reads the database again.';
		}
	});
</script>

{#snippet actions()}
	<Button variant="subtle" onclick={() => (open = false)} disabled={busy}>Cancel</Button>
	<Button type="submit" loading={save.isPending} disabled={busy || !parsed || !ttlValid}>
		Save value
	</Button>
{/snippet}

<FullscreenDialog
	bind:open
	actions={editable ? actions : undefined}
	title={key?.name ?? 'Cached key'}
	meta={key ? kinds[key.kind].label : undefined}
	onsubmit={submit}
>
	{#if value.isError}
		<Alert>{messageOf(value.error, 'Could not read this key')}</Alert>
	{:else if current}
		<FormSection title="Key" description="Where Redis keeps it, without this server’s prefix.">
			<Input label="Name" value={current.name} readOnly copyable />
			<FieldGrid>
				<Input label="Group" value={current.group || 'N/A'} readOnly />
				<Input label="Entry" value={current.entry || 'N/A'} readOnly />
				<Input label="Expires in" value={formatTTL(current.ttl_seconds)} readOnly />
				<Input
					label="Size"
					value={current.size_bytes > 0 ? formatBytes(current.size_bytes) : 'unknown'}
					readOnly
				/>
			</FieldGrid>
			<span class="tags">
				<Tag small tone={kinds[current.kind].tone}>{kinds[current.kind].label}</Tag>
				<Tag small>{current.type}</Tag>
				{#if !editable}<Tag small>read only</Tag>{/if}
			</span>
		</FormSection>

		<FormSection
			title="Value"
			description={editable
				? 'What the next reader decodes. A value that no longer fits what the server expects is read as missing, and the database is read instead.'
				: 'Shown as it is. What decides who is signed in, and a generation counter, are never edited by hand.'}
		>
			<CodeEditor
				label="JSON"
				bind:value={text}
				rows={16}
				readOnly={!editable}
				error={editable && !parsed ? 'This is not JSON, or it is null.' : undefined}
			/>
			{#if editable}
				<Input
					label="Keep for"
					icon={RiTimerLine}
					bind:value={ttl}
					inputmode="numeric"
					suffix="seconds"
					placeholder="as long as it had left"
					hint="Up to a day. Left empty, the value keeps the time it had left."
					error={ttlValid ? undefined : 'A whole number of seconds, up to 86400.'}
				/>
			{/if}
		</FormSection>

		{#if current.kind !== 'generation'}
			<DangerZone title="Remove this key" description={consequence}>
				{#if confirmingDelete}
					<Button
						variant="subtle"
						size="sm"
						onclick={() => (confirmingDelete = false)}
						disabled={busy}
					>
						Keep it
					</Button>
					<Button
						colorPalette="danger"
						size="sm"
						loading={remove.isPending}
						disabled={busy}
						onclick={() => remove.mutate()}
					>
						Remove this key
					</Button>
				{:else}
					<Button
						colorPalette="danger"
						variant="subtle"
						size="sm"
						onclick={() => (confirmingDelete = true)}
						disabled={busy}
					>
						<Icon icon={RiDeleteBinLine} />
						Remove
					</Button>
				{/if}
			</DangerZone>
		{:else}
			<Note>A generation counter is not removed: clear its group instead.</Note>
		{/if}
	{:else}
		<Note>Reading…</Note>
	{/if}
</FullscreenDialog>

<style>
	.tags {
		display: inline-flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-1);
	}
</style>
