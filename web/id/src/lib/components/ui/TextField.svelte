<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';
	import { Input } from 'ui';

	type Props = Omit<HTMLInputAttributes, 'value' | 'size'> & {
		label: string;
		value: string;
		hint?: string;
		error?: string;
	};

	let { label, value = $bindable(''), hint, error, ...input }: Props = $props();

	// Passed on as they came; the shared Input types them where they land.
	const attributes = $derived(input as Record<string, unknown>);
</script>

<Input {label} bind:value {hint} {error} {...attributes} />
