<script lang="ts">
	import type { SSODomainProof } from '#lib/api/index.js';
	import { Badge, Button, Input, List, ListItem } from '#lib/components/ui/index.js';

	type Props = {
		proofs: SSODomainProof[];
		canWrite: boolean;
		/** The domain being checked now, if any. */
		checking?: string;
		oncheck: (domain: string) => void;
	};

	let { proofs, canWrite, checking, oncheck }: Props = $props();

	const status = (proof: SSODomainProof) => (proof.proved ? 'Proved' : 'Not proved');
</script>

<List bordered label="Domain proofs">
	{#each proofs as proof (proof.domain)}
		<ListItem title={proof.domain}>
			{#if !proof.proved}
				<Input label="TXT record name" value={proof.record_name} readOnly copyable />
				<Input label="TXT record value" value={proof.record_value} readOnly copyable />
			{/if}
			{#snippet end()}
				<Badge tone={proof.proved ? 'success' : 'neutral'}>{status(proof)}</Badge>
				{#if canWrite && !proof.proved}
					<Button
						variant="subtle"
						loading={checking === proof.domain}
						onclick={() => oncheck(proof.domain)}
					>
						Check
					</Button>
				{/if}
			{/snippet}
		</ListItem>
	{/each}
</List>
