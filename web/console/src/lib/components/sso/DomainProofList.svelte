<script lang="ts">
	import type { SSODomainProof } from '#lib/api/index.js';
	import { Alert, Badge, Button, Input, List, ListItem } from '#lib/components/ui/index.js';

	type Props = {
		proofs: SSODomainProof[];
		canWrite: boolean;
		/** The domain being checked now, if any. */
		checking?: string;
		oncheck: (domain: string) => void;
	};

	let { proofs, canWrite, checking, oncheck }: Props = $props();

	function status(proof: SSODomainProof): string {
		if (proof.carried_over) return 'Carried over, not proved';
		return proof.proved ? 'Proved' : 'Not proved';
	}

	function tone(proof: SSODomainProof): 'success' | 'warning' | 'neutral' {
		if (proof.carried_over) return 'warning';
		return proof.proved ? 'success' : 'neutral';
	}

	/** Carried over, it links without anybody having proved it, so it is worth proving too. */
	const provable = (proof: SSODomainProof) => !proof.proved || proof.carried_over;
</script>

{#if proofs.some((proof) => proof.carried_over)}
	<Alert tone="warning">
		Some domains link to existing accounts only because this connection listed them before domains
		were proved. Nobody has shown they are the organisation's: publish their TXT records and check
		them, or remove the domains.
	</Alert>
{/if}

<List bordered label="Domain proofs">
	{#each proofs as proof (proof.domain)}
		<ListItem title={proof.domain}>
			{#if provable(proof)}
				<Input label="TXT record name" value={proof.record_name} readOnly copyable />
				<Input label="TXT record value" value={proof.record_value} readOnly copyable />
			{/if}
			{#snippet end()}
				<Badge tone={tone(proof)}>{status(proof)}</Badge>
				{#if canWrite && provable(proof)}
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
