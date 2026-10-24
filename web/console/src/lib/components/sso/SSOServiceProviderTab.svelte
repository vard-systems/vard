<script lang="ts">
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { ssoApi, type SSOConnection } from '#lib/api/index.js';
	import { Alert, Button, FormSection, Input, Note, notify } from '#lib/components/ui/index.js';
	import { keys } from '#lib/query/index.js';
	import { saveFile } from '#lib/utils/download.js';

	/** A certificate this close to expiring is worth a warning. */
	const EXPIRY_WARNING = 30 * 24 * 3600 * 1000;

	type Props = {
		current: SSOConnection | null;
		canWrite: boolean;
		/** The connection as the server has it after its metadata was read again. */
		onrefreshed: (connection: SSOConnection) => void;
	};

	let { current, canWrite, onrefreshed }: Props = $props();

	const queryClient = useQueryClient();

	const expiring = $derived.by(() => {
		const expires = current?.identity_provider?.certificate_expires;
		if (!expires) return null;

		const when = new Date(expires);
		return when.getTime() - Date.now() < EXPIRY_WARNING ? when : null;
	});

	const refresh = createMutation(() => ({
		mutationFn: () => ssoApi.refreshMetadata(current?.id ?? ''),
		onSuccess: async ({ connection: saved }) => {
			notify.success('Metadata read again');
			onrefreshed(saved);
			await queryClient.invalidateQueries({ queryKey: keys.sso.all });
		},
		onError: (err: unknown) => {
			notify.error(err, 'Could not read the metadata again');
		}
	}));

	function downloadCertificate(connection: SSOConnection) {
		saveFile(
			`${connection.slug}-certificate.pem`,
			connection.service_provider.certificate ?? '',
			'application/x-pem-file'
		);
	}
</script>

{#if !current}
	<Note>Create the connection to get the addresses its provider needs.</Note>
{:else if current.protocol === 'oidc'}
	<FormSection
		title="What to give the provider"
		description="Register this redirect URI with the application at the provider."
	>
		<Input
			label="Redirect URI"
			value={current.service_provider.callback_url ?? ''}
			readOnly
			copyable
		/>
	</FormSection>
{:else}
	<FormSection
		title="What to give the provider"
		description="Give the provider the metadata URL — or, if it asks for them one by one, these."
	>
		<Input
			label="ACS URL (reply URL)"
			value={current.service_provider.acs_url ?? ''}
			readOnly
			copyable
		/>
		<Input
			label="Entity ID (audience)"
			value={current.service_provider.entity_id ?? ''}
			readOnly
			copyable
		/>
		<Input
			label="Service provider metadata"
			value={current.service_provider.metadata_url ?? ''}
			readOnly
			copyable
		/>

		<div>
			<Button variant="subtle" size="sm" onclick={() => downloadCertificate(current)}>
				<i class="ri-download-2-line" aria-hidden="true"></i>
				Download signing certificate
			</Button>
		</div>
	</FormSection>

	{#if current.identity_provider}
		<FormSection title="The identity provider" description="What its metadata says about it.">
			<dl class="facts">
				<dt>Entity ID</dt>
				<dd><code>{current.identity_provider.entity_id}</code></dd>
				<dt>Sign-in URL</dt>
				<dd><code>{current.identity_provider.sso_url}</code></dd>
				{#if current.identity_provider.certificate_expires}
					<dt>Certificate expires</dt>
					<dd>
						{new Date(current.identity_provider.certificate_expires).toLocaleDateString()}
					</dd>
				{/if}
			</dl>

			{#if expiring}
				<Alert tone="warning">
					{`Its certificate expires on ${expiring.toLocaleDateString()}. Refresh the metadata once the provider has rolled it over.`}
				</Alert>
			{/if}

			{#if canWrite && current.metadata_url}
				<div>
					<Button
						variant="subtle"
						size="sm"
						onclick={() => refresh.mutate()}
						loading={refresh.isPending}
					>
						<i class="ri-refresh-line" aria-hidden="true"></i>
						Refresh metadata
					</Button>
				</div>
			{/if}
		</FormSection>
	{/if}
{/if}

<style>
	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: var(--space-1) var(--space-3);
		margin: 0;
		font-size: var(--text-sm);
	}

	.facts dt {
		color: var(--color-text-hint);
	}

	.facts dd {
		margin: 0;
		overflow-wrap: anywhere;
	}

	.facts code {
		font-family: var(--font-mono);
	}
</style>
