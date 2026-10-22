<script lang="ts">
	import { ApiError, socialApi } from '#lib/api/index.js';
	import { Button, CopyButton, Input, Textarea } from '#lib/components/ui/index.js';

	type Props = {
		providerId?: string | null;
		stored: boolean;
		label: string;
		hint?: string;
		value: string;
		multiline?: boolean;
		disabled?: boolean;
	};

	let {
		providerId = null,
		stored,
		label,
		hint,
		value = $bindable(''),
		multiline = false,
		disabled = false
	}: Props = $props();

	type Mode = 'stored' | 'revealed' | 'replacing';

	let mode = $state<Mode>('stored');
	let secret = $state('');
	let loading = $state(false);
	let error = $state('');

	const showing = $derived(stored && providerId ? mode : 'replacing');

	async function reveal() {
		if (!providerId) return;

		error = '';
		loading = true;

		try {
			({ secret } = await socialApi.secret(providerId));
			mode = 'revealed';
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Could not read this secret';
		} finally {
			loading = false;
		}
	}

	function hide() {
		// Not kept around any longer than it is on screen.
		secret = '';
		mode = 'stored';
	}

	function replace() {
		secret = '';
		mode = 'replacing';
	}

	function keep() {
		value = '';
		mode = 'stored';
	}
</script>

<div class="secret">
	{#if showing === 'replacing'}
		{#if multiline}
			<Textarea
				{label}
				bind:value
				rows={4}
				{disabled}
				placeholder="-----BEGIN PRIVATE KEY-----"
				hint={hint ?? 'Encrypted here, and never sent anywhere else.'}
			/>
		{:else}
			<Input
				{label}
				bind:value
				type="password"
				{disabled}
				autocomplete="off"
				hint={hint ?? 'Encrypted here, and never sent anywhere else.'}
			/>
		{/if}

		{#if stored}
			<div class="actions">
				<Button size="sm" variant="subtle" onclick={keep}>Keep the stored one</Button>
			</div>
		{/if}
	{:else}
		<Input
			{label}
			value={showing === 'revealed' ? secret : '••••••••••••••••'}
			readOnly
			hint={showing === 'revealed'
				? 'Reading this was recorded in the activity log.'
				: (hint ?? 'Stored and encrypted. Reveal it to check it against the provider.')}
		/>

		<div class="actions">
			{#if showing === 'revealed'}
				<span class="copy"><CopyButton value={secret} {label} /></span>
				<Button size="sm" variant="subtle" onclick={hide}>
					<i class="ri-eye-off-line" aria-hidden="true"></i>
					Hide
				</Button>
			{:else}
				<Button size="sm" variant="subtle" {loading} disabled={loading} onclick={reveal}>
					<i class="ri-eye-line" aria-hidden="true"></i>
					Reveal
				</Button>
			{/if}

			<Button size="sm" variant="subtle" onclick={replace}>
				<i class="ri-pencil-line" aria-hidden="true"></i>
				Replace
			</Button>
		</div>
	{/if}

	{#if error}
		<p class="error">{error}</p>
	{/if}
</div>

<style>
	.secret {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.actions {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.copy {
		display: inline-flex;
	}

	.error {
		margin: 0;
		color: var(--color-danger);
		font-size: var(--text-sm);
	}
</style>
