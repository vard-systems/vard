<script lang="ts">
	import { RiErrorWarningLine } from 'svelte-remixicon';
	import { CopyButton, Icon } from '$lib/components/ui';

	type Props = {
		clientId: string;
		secret: string;
	};

	let { clientId, secret }: Props = $props();

	let panel: HTMLElement;

	$effect(() => {
		void secret;
		panel.scrollIntoView({ behavior: 'smooth', block: 'start' });
	});
</script>

<!-- The one time the secret exists outside the application: only its hash is
     kept, so the panel says so plainly and makes copying it easy. -->
<div class="panel" role="status" bind:this={panel}>
	<div class="head">
		<Icon icon={RiErrorWarningLine} size="1.125rem" />
		<div>
			<strong>Copy the client secret now</strong>
			<p>It is shown this once. If it is lost, rotate it and give the application the new one.</p>
		</div>
	</div>

	<dl>
		<dt>Client ID</dt>
		<dd>
			<code>{clientId}</code>
			<CopyButton value={clientId} label="client ID" />
		</dd>

		<dt>Client secret</dt>
		<dd>
			<code>{secret}</code>
			<CopyButton value={secret} label="client secret" />
		</dd>
	</dl>
</div>

<style>
	.panel {
		scroll-margin-top: var(--space-5);
		margin-bottom: var(--space-5);
		padding: var(--space-3) var(--space-4);
		border: 1px solid color-mix(in srgb, var(--color-success), transparent 60%);
		border-radius: var(--radius-surface);
		background: var(--surface-success);
	}

	.head {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2);
		color: var(--color-success);
	}

	.head strong {
		color: var(--color-text);
	}

	.head p {
		margin: 2px 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	dl {
		display: grid;
		grid-template-columns: 7rem 1fr;
		align-items: center;
		gap: var(--space-2) var(--space-3);
		margin: var(--space-3) 0 0;
	}

	dt {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	dd {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		min-width: 0;
		margin: 0;
		padding: 2px 2px 2px 10px;
		border-radius: var(--radius-sm);
		background: var(--color-surface);
	}

	code {
		flex: 1;
		overflow-wrap: anywhere;
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		user-select: all;
	}

	@media (max-width: 30rem) {
		dl {
			grid-template-columns: 1fr;
		}
	}
</style>
