<script lang="ts">
	import { RiCheckLine, RiFileCopyLine } from '#lib/icons.js';
	import IconButton from './IconButton.svelte';

	type Props = {
		value: string;
		label?: string;
	};

	let { value, label = 'value' }: Props = $props();

	let copied = $state(false);
	let timer: ReturnType<typeof setTimeout>;

	async function copy() {
		await navigator.clipboard.writeText(value);

		copied = true;
		clearTimeout(timer);
		timer = setTimeout(() => (copied = false), 1500);
	}
</script>

<IconButton
	icon={copied ? RiCheckLine : RiFileCopyLine}
	label={copied ? 'Copied' : `Copy ${label}`}
	size="sm"
	placement="left"
	onclick={copy}
/>
