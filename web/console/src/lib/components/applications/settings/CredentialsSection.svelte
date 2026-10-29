<script lang="ts">
	import type { Application } from '#lib/api/index.js';
	import { Button, FormSection, Input, Note, Select } from '#lib/components/ui/index.js';
	import { formatDateTime } from '#lib/utils/format.js';
	import { authMethods } from '../applications';
	import type { ApplicationForm } from '../form.svelte';

	type Props = {
		form: ApplicationForm;
		current: Application | null;
		editable: boolean;
		saving: boolean;
		onrotate: () => void;
	};

	let { form, current, editable, saving, onrotate }: Props = $props();

	/** Only the secret's last characters are kept, so that is all there is to show. */
	const maskedSecret = $derived(
		current?.has_secret ? `••••••••••••••••${current.secret_hint}` : 'No secret yet'
	);

	const method = $derived(form.input.token_auth_method);
	const usesSecret = $derived(method === 'client_secret_basic' || method === 'client_secret_post');
</script>

<FormSection
	title="Credentials"
	description={form.kind.public
		? 'A public client cannot keep a secret, so it has none and must use PKCE.'
		: 'What the application proves who it is with at the token endpoint.'}
>
	{#if current}
		<Input label="Client ID" value={current.client_id} readOnly copyable />
	{:else}
		<Note boxed>
			{usesSecret
				? 'The client ID and secret are made when you create the application.'
				: 'The client ID is made when you create the application.'}
		</Note>
	{/if}

	{#if !form.kind.public}
		<Select
			label="Client authentication"
			bind:value={form.input.token_auth_method}
			options={authMethods}
		/>

		{#if current && usesSecret}
			<div class="secret">
				<Input
					label="Client secret"
					value={maskedSecret}
					readOnly
					hint={current.secret_created_at
						? `Created ${formatDateTime(current.secret_created_at)}. Only its last characters are kept.`
						: undefined}
				/>

				{#if editable}
					<div class="rotate">
						<Button size="sm" variant="subtle" onclick={onrotate} disabled={saving}>
							<i class="ri-refresh-line" aria-hidden="true"></i>
							Rotate secret
						</Button>
					</div>
				{/if}
			</div>
		{/if}
	{/if}
</FormSection>

<style>
	.secret {
		display: grid;
		grid-template-columns: 1fr auto;
		align-items: start;
		gap: var(--space-3);
	}

	.secret :global([data-part='input']) {
		font-family: var(--font-mono);
		letter-spacing: 0.02em;
	}

	.rotate {
		display: flex;
		flex-direction: column;
		align-items: flex-end;
		gap: var(--space-2);
		padding-top: 14px;
	}

	@media (max-width: 36rem) {
		.secret {
			grid-template-columns: 1fr;
		}

		.rotate {
			align-items: flex-start;
			padding-top: 0;
		}
	}
</style>
