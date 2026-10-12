<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { RiAddLine, RiDeleteBinLine, RiPencilLine } from 'svelte-remixicon';
	import { usersApi, type FieldType, type UserField } from '$lib/api';
	import {
		Badge,
		Button,
		FullscreenDialog,
		FormSection,
		Icon,
		IconButton,
		Input,
		Select,
		Switch,
		notify
	} from '$lib/components/ui';
	import type { SelectOption } from '$lib/components/ui';
	import { keys } from '$lib/query';
	import { additional, builtins } from './fields';
	import { fieldIcons } from './fieldIcons';

	type Props = {
		fields: UserField[];
		open: boolean;
	};

	let { fields, open = $bindable(false) }: Props = $props();

	const builtinFields = $derived(builtins(fields));
	const addedFields = $derived(additional(fields));

	const queryClient = useQueryClient();

	const refill = () => queryClient.invalidateQueries({ queryKey: keys.users.all });

	const types: SelectOption<FieldType>[] = [
		{ value: 'text', label: 'Plain text', icon: fieldIcons.text },
		{ value: 'number', label: 'Number', icon: fieldIcons.number },
		{ value: 'bool', label: 'Bool', icon: fieldIcons.bool },
		{ value: 'email', label: 'Email', icon: fieldIcons.email },
		{ value: 'date', label: 'Date', icon: fieldIcons.date }
	];

	let editing = $state<UserField | null>(null);

	let name = $state('');
	let label = $state('');
	let type = $state<FieldType>('text');
	let required = $state(false);
	let unique = $state(false);
	let min = $state('');
	let max = $state('');
	let startsWith = $state('');

	let busy = $state(false);

	const bounded = $derived(type === 'number' || type === 'text' || type === 'email');
	const prefixed = $derived(type === 'text' || type === 'email');
	const lengths = $derived(type !== 'number');

	function suggestName(from: string): string {
		return from
			.trim()
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '_')
			.replace(/^[^a-z]+/, '')
			.replace(/_+$/, '')
			.slice(0, 64);
	}

	function blank() {
		editing = null;
		name = '';
		label = '';
		type = 'text';
		required = false;
		unique = false;
		min = '';
		max = '';
		startsWith = '';
	}

	function edit(field: UserField) {
		editing = field;
		name = field.name;
		label = field.label;
		type = field.type;
		required = field.is_required;
		unique = field.is_unique;
		min = field.min === null ? '' : String(field.min);
		max = field.max === null ? '' : String(field.max);
		startsWith = field.starts_with;
	}

	function bound(value: string): number | null {
		const trimmed = value.trim();
		if (trimmed === '') return null;

		const parsed = Number(trimmed);
		return Number.isFinite(parsed) ? parsed : null;
	}

	function rules() {
		return {
			label: label.trim() || name.trim(),
			is_required: required,
			is_unique: unique,
			min: bounded ? bound(min) : null,
			max: bounded ? bound(max) : null,
			starts_with: prefixed ? startsWith.trim() : ''
		};
	}

	const save = createMutation(() => ({
		mutationFn: () =>
			editing?.id
				? usersApi.updateField(editing.id, rules())
				: usersApi.addField({ ...rules(), name: name.trim() || suggestName(label), type }),
		onSuccess: async () => {
			notify.success(editing?.id ? 'Field saved' : 'Field added');
			await refill();
			blank();
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not save this field');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	const remove = createMutation(() => ({
		mutationFn: (field: UserField) => usersApi.removeField(field.id ?? ''),
		onSuccess: async (_result: void, field: UserField) => {
			notify.success(`${field.label} removed`);
			await refill();

			if (editing?.id === field.id) blank();
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not remove this field');
		},
		onSettled: () => {
			busy = false;
		}
	}));

	function submit(event: SubmitEvent) {
		event.preventDefault();

		if (busy) return;

		busy = true;
		save.mutate();
	}

	function summary(field: UserField): string[] {
		const said: string[] = [];

		if (field.is_required) said.push('required');
		if (field.is_unique) said.push('unique');

		const unit = field.type === 'number' ? '' : ' chars';
		if (field.min !== null && field.max !== null) said.push(`${field.min}–${field.max}${unit}`);
		else if (field.min !== null) said.push(`min ${field.min}${unit}`);
		else if (field.max !== null) said.push(`max ${field.max}${unit}`);

		if (field.starts_with) said.push(`starts with ${field.starts_with}`);

		return said;
	}
</script>

<FullscreenDialog
	bind:open
	title="User fields"
	description="The columns a user record has, and the rules their values keep."
>
	<FormSection
		title="Built-in fields"
		description="Every user record has these. They are columns of the record itself, so they cannot be changed or removed."
	>
		<ul class="fields">
			{#each builtinFields as field (field.name)}
				<li class="locked">
					<Icon icon={fieldIcons[field.type]} />

					<span class="name">{field.name}</span>
					<span class="hint">{field.type}</span>

					<span class="rules">
						{#each summary(field) as rule (rule)}
							<Badge>{rule}</Badge>
						{/each}
					</span>
				</li>
			{/each}
		</ul>
	</FormSection>

	<FormSection
		title="Added fields"
		description="Anything else this organisation keeps about a user. Adding one needs no migration."
	>
		<ul class="fields">
			{#each addedFields as field (field.id)}
				<li class:editing={editing?.id === field.id}>
					<Icon icon={fieldIcons[field.type]} />

					<span class="name">{field.name}</span>
					<span class="hint">{field.type}</span>

					<span class="rules">
						{#each summary(field) as rule (rule)}
							<Badge>{rule}</Badge>
						{/each}
					</span>

					<IconButton
						icon={RiPencilLine}
						label="Edit {field.name}"
						size="sm"
						onclick={() => edit(field)}
						disabled={busy}
					/>

					<IconButton
						icon={RiDeleteBinLine}
						label="Remove {field.name}"
						size="sm"
						colorPalette="danger"
						onclick={() => {
							if (busy) return;
							busy = true;
							remove.mutate(field);
						}}
						disabled={busy}
					/>
				</li>
			{:else}
				<li class="hint">No added fields yet.</li>
			{/each}
		</ul>
	</FormSection>

	<FormSection
		title={editing ? `Edit ${editing.name}` : 'Add a field'}
		description={editing
			? 'A field keeps its name and type once records hold values under them.'
			: 'The name is what the value is stored under; the type cannot change later.'}
	>
		<form onsubmit={submit}>
			<div class="row">
				<Input
					label="Label"
					value={label}
					oninput={(event) => {
						label = event.currentTarget.value;
						if (!editing) name = suggestName(label);
					}}
					placeholder="Phone number"
				/>

				<Input
					label="Name"
					bind:value={name}
					placeholder="phone_number"
					readOnly={editing !== null}
				/>

				<Select label="Type" bind:value={type} options={types} readOnly={editing !== null} />
			</div>

			{#if bounded || prefixed}
				<div class="row">
					{#if bounded}
						<Input
							label={lengths ? 'Least characters' : 'Smallest value'}
							bind:value={min}
							type="number"
							placeholder="any"
						/>

						<Input
							label={lengths ? 'Most characters' : 'Largest value'}
							bind:value={max}
							type="number"
							placeholder="any"
						/>
					{/if}

					{#if prefixed}
						<Input label="Must start with" bind:value={startsWith} placeholder="+" />
					{/if}
				</div>
			{/if}

			<div class="switches">
				<Switch label="Required" bind:checked={required} />
				<Switch label="Unique" bind:checked={unique} />
			</div>

			<div class="actions">
				{#if editing}
					<Button variant="subtle" onclick={blank} disabled={busy}>Cancel</Button>
				{/if}

				<Button type="submit" disabled={busy || (name.trim() === '' && label.trim() === '')}>
					{#if !editing}
						<Icon icon={RiAddLine} />
					{/if}
					{editing ? 'Save field' : 'Add field'}
				</Button>
			</div>
		</form>
	</FormSection>

	{#snippet actions()}
		<Button variant="subtle" onclick={() => (open = false)}>Done</Button>
	{/snippet}
</FullscreenDialog>

<style>
	.fields {
		display: flex;
		flex-direction: column;
		gap: 2px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.fields li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2);
		border-radius: var(--radius-sm);
		font-size: var(--text-base);
	}

	.fields li:hover {
		background: var(--row-hover);
	}

	.fields li.locked {
		opacity: 0.75;
	}

	.fields li.locked:hover {
		background: transparent;
	}

	.fields li.editing {
		background: var(--surface-info);
	}

	.name {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.hint {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.rules {
		flex: 1;
		display: flex;
		flex-wrap: wrap;
		justify-content: flex-end;
		gap: var(--space-1);
	}

	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.row {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
		gap: var(--space-2);
	}

	.switches {
		display: flex;
		gap: var(--space-5);
	}

	.actions {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
	}
</style>
