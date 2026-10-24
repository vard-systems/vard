<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLFormAttributes } from 'svelte/elements';

	type Props = Omit<HTMLFormAttributes, 'onsubmit'> & {
		/** Called instead of the browser's own submission: every form here
		    talks to the API with JSON, never with a form post. */
		onsubmit: () => unknown;
		children: Snippet;
	};

	let { onsubmit, children, ...rest }: Props = $props();

	function submit(event: SubmitEvent) {
		event.preventDefault();
		void onsubmit();
	}
</script>

<form novalidate onsubmit={submit} {...rest}>
	{@render children()}
</form>

<style>
	form {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}
</style>
