<script lang="ts">
	import { organizationApi, type Organization } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import { Button, notify } from '#lib/components/ui/index.js';
	import { ORGANIZATION_LOGO } from '#lib/constants.js';
	import { logoProblem } from './settings';

	type Props = {
		/** The saved logo's version, "" while there is none. */
		version: string;
		editable: boolean;
		onsaved: (organization: Organization) => void;
	};

	let { version, editable, onsaved }: Props = $props();

	const id = $props.id();

	let file: HTMLInputElement | undefined = $state();
	let busy = $state<'upload' | 'remove' | null>(null);

	const src = $derived(organizationApi.logoUrl(version) || BRAND.logo);

	async function upload(event: Event & { currentTarget: HTMLInputElement }) {
		const chosen = event.currentTarget.files?.[0];
		// The same file chosen again after a refusal still fires a change.
		event.currentTarget.value = '';
		if (!chosen) return;

		const problem = logoProblem(chosen);
		if (problem) {
			notify.problem(problem);
			return;
		}

		busy = 'upload';
		try {
			const result = await organizationApi.uploadLogo(chosen);
			onsaved(result.organization);
			notify.success('Logo saved', 'The header and the sign-in pages show it now.');
		} catch (err) {
			notify.error(err, 'Could not save the logo');
		} finally {
			busy = null;
		}
	}

	async function remove() {
		busy = 'remove';
		try {
			const result = await organizationApi.removeLogo();
			onsaved(result.organization);
			notify.success('Logo removed', 'The panel shows its own in its place.');
		} catch (err) {
			notify.error(err, 'Could not remove the logo');
		} finally {
			busy = null;
		}
	}
</script>

<div class="logo-field" role="group" aria-labelledby="{id}-label">
	<div class="preview">
		<img {src} alt={version ? 'App logo' : "App logo: the panel's own"} />
	</div>

	<div class="text">
		<span class="label" id="{id}-label">
			App logo
			{#if !version}<span class="default">Default</span>{/if}
		</span>
		<span class="hint">
			A square logo for the header and the sign-in pages. PNG or JPEG, up to 512 KB.
		</span>

		{#if editable}
			<div class="actions">
				<input
					id="{id}-file"
					bind:this={file}
					type="file"
					accept={ORGANIZATION_LOGO.accept}
					aria-label="Choose an app logo"
					hidden
					onchange={upload}
				/>
				<Button
					size="sm"
					variant="subtle"
					loading={busy === 'upload'}
					disabled={busy !== null}
					onclick={() => file?.click()}
				>
					<i class="ri-upload-2-line" aria-hidden="true"></i>
					{version ? 'Replace' : 'Upload'}
				</Button>
				{#if version}
					<Button
						size="sm"
						variant="ghost"
						colorPalette="danger"
						loading={busy === 'remove'}
						disabled={busy !== null}
						onclick={remove}
					>
						Remove
					</Button>
				{/if}
			</div>
		{/if}
	</div>
</div>

<style>
	.logo-field {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		min-width: 0;
	}

	.preview {
		display: grid;
		flex-shrink: 0;
		place-items: center;
		width: 72px;
		height: 72px;
		padding: var(--space-2);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface-alt);
	}

	.preview img {
		width: 100%;
		height: 100%;
		object-fit: contain;
	}

	.text {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.label {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--color-text);
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.default {
		color: var(--color-text-hint);
		font-size: 10px;
		font-weight: 600;
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.hint {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.actions {
		display: flex;
		gap: var(--space-1);
		margin-top: var(--space-1);
	}
</style>
