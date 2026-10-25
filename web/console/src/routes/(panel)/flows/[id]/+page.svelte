<script lang="ts">
	import { pageTitle } from '#lib/brand.js';
	import FlowEditor from '#lib/components/flows/editor/FlowEditor.svelte';
	import { can } from '#lib/permissions.js';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head><title>{pageTitle(data.flow.name, 'Login flows')}</title></svelte:head>

<!-- Keyed, so moving from one flow to another — after Duplicate — starts a
     fresh editor rather than carrying the last one's draft across. -->
{#key data.flow.id}
	<FlowEditor
		flow={data.flow}
		initial={data.flow}
		kinds={data.stepKinds}
		taken={data.taken}
		editable={can(data.admin, 'login_flows.write')}
	/>
{/key}
