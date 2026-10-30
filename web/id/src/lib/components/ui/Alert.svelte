<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Alert } from 'ui';

	type Props = {
		tone?: 'danger' | 'success' | 'info' | 'warning';
		children: Snippet;
	};

	let { tone = 'danger', children }: Props = $props();
</script>

<Alert {tone}>{@render children()}</Alert>
