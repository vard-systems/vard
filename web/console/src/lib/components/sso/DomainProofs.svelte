<script lang="ts">
	import { createMutation, createQuery, useQueryClient } from '@tanstack/svelte-query';
	import { ssoApi } from '#lib/api/index.js';
	import { Note, notify } from '#lib/components/ui/index.js';
	import { keys, ssoDomainsOptions } from '#lib/query/index.js';
	import DomainProofList from './DomainProofList.svelte';

	type Props = {
		connectionId: string;
		canWrite: boolean;
	};

	let { connectionId, canWrite }: Props = $props();

	const queryClient = useQueryClient();
	const proofs = createQuery(() => ssoDomainsOptions(connectionId));

	const check = createMutation(() => ({
		mutationFn: (domain: string) => ssoApi.checkDomain(connectionId, domain),
		onSuccess: async ({ domain }) => {
			if (domain.proved) notify.success(`${domain.domain} is proved`);
			else notify.error(null, `No matching TXT record at ${domain.record_name} yet`);
			await queryClient.invalidateQueries({ queryKey: keys.sso.domains(connectionId) });
		},
		onError: (err: unknown) => notify.error(err, 'Could not check the domain')
	}));
</script>

<Note>
	The connection makes new accounts at any of its domains, but links to an account that already
	exists only at a proved one. Publish the TXT record, then check it.
</Note>

{#if proofs.data && proofs.data.length > 0}
	<DomainProofList
		proofs={proofs.data}
		{canWrite}
		checking={check.isPending ? check.variables : undefined}
		oncheck={(domain) => check.mutate(domain)}
	/>
{/if}
