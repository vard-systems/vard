<script lang="ts">
	import { Button } from '#lib/components/ui/index.js';

	type Props = {
		yes: string;
		no: string;
		/** Which answer is on its way to the server, so only that one spins. */
		sending?: 'yes' | 'no' | null;
		onanswer: (yes: boolean) => unknown;
	};

	let { yes, no, sending = null, onanswer }: Props = $props();
</script>

<div class="decision">
	<Button
		block
		loading={sending === 'yes'}
		disabled={sending !== null}
		onclick={() => onanswer(true)}
	>
		{yes}
	</Button>
	<Button
		block
		variant="secondary"
		loading={sending === 'no'}
		disabled={sending !== null}
		onclick={() => onanswer(false)}
	>
		{no}
	</Button>
</div>

<style>
	.decision {
		display: grid;
		gap: var(--space-2);
		margin-top: var(--space-1);
	}
</style>
