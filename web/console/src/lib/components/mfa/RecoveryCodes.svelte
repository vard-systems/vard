<script lang="ts">
	import { RiCheckLine, RiFileCopyLine } from 'svelte-remixicon';
	import { Alert, Button, Checkbox } from '$lib/components/ui';

	type Props = {
		codes: string[];
		doneLabel?: string;
		onDone: () => void;
	};

	let { codes, doneLabel = 'Done', onDone }: Props = $props();

	let saved = $state(false);
	let copied = $state(false);

	async function copy() {
		await navigator.clipboard.writeText(codes.join('\n'));
		copied = true;
		setTimeout(() => (copied = false), 1500);
	}
</script>

<div class="codes-step">
	<Alert tone="warning">
		Save these recovery codes somewhere safe, such as a password manager. Each one signs you in once
		if you lose your phone. This is the only time they are shown.
	</Alert>

	<ol class="codes" aria-label="Recovery codes">
		{#each codes as code (code)}
			<li><code>{code}</code></li>
		{/each}
	</ol>

	<div class="actions">
		<Button variant="subtle" icon={copied ? RiCheckLine : RiFileCopyLine} onclick={copy}>
			{copied ? 'Copied' : 'Copy all'}
		</Button>
	</div>

	<Checkbox
		checked={saved}
		onChange={(value) => (saved = value)}
		label="I have saved my recovery codes"
	/>

	<Button size="lg" disabled={!saved} onclick={onDone}>{doneLabel}</Button>
</div>

<style>
	.codes-step {
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
	}

	.codes {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: var(--space-2);
		margin: 0;
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		background: var(--color-surface-alt);
		list-style: none;
	}

	.codes code {
		font-family: var(--font-mono);
		font-size: var(--text-lg);
		letter-spacing: 0.04em;
	}

	.actions {
		display: flex;
		justify-content: flex-end;
	}
</style>
