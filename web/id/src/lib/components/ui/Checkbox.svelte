<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Checkbox } from 'ui';

	type Props = {
		checked: boolean;
		disabled?: boolean;
		id?: string;
		name?: string;
		children: Snippet;
	};

	let { checked = $bindable(false), disabled, id, name, children }: Props = $props();
</script>

<Checkbox {checked} onChange={(value) => (checked = value)} {disabled} {id} {name}>
	{@render children()}
</Checkbox>
