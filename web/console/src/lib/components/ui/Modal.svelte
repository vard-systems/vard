<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Dialog } from '@ark-ui/svelte/dialog';
	import { Portal } from '@ark-ui/svelte/portal';
	import { RiCloseLine } from 'svelte-remixicon';
	import IconButton from './IconButton.svelte';

	type Props = {
		open?: boolean;
		title: string;
		description?: string;
		size?: 'sm' | 'md' | 'lg';
		closable?: boolean;
		children?: Snippet;
		footer?: Snippet;
	};

	let {
		open = $bindable(false),
		title,
		description,
		size = 'md',
		closable = true,
		children,
		footer
	}: Props = $props();

	/** Where focus lands when it opens: the heading, which a screen reader
	    reads first, rather than the close button, whose tooltip would pop up
	    over the title before anything was done. */
	let heading = $state<HTMLElement | null>(null);
</script>

<Dialog.Root
	{open}
	onOpenChange={(details) => {
		if (closable || details.open) open = details.open;
	}}
	lazyMount
	unmountOnExit
	initialFocusEl={() => heading}
>
	<Portal>
		<Dialog.Backdrop />
		<Dialog.Positioner>
			<Dialog.Content class="modal-content" data-size={size}>
				<header bind:this={heading} tabindex="-1">
					<div>
						<Dialog.Title>{title}</Dialog.Title>
						{#if description}<Dialog.Description>{description}</Dialog.Description>{/if}
					</div>

					{#if closable}
						<IconButton
							icon={RiCloseLine}
							label="Close"
							size="sm"
							placement="left"
							onclick={() => (open = false)}
						/>
					{/if}
				</header>

				{#if children}<div class="body">{@render children()}</div>{/if}

				{#if footer}<footer>{@render footer()}</footer>{/if}
			</Dialog.Content>
		</Dialog.Positioner>
	</Portal>
</Dialog.Root>

<style>
	:global(.modal-content[data-scope='dialog'][data-part='content']) {
		--dialog-width: 40rem;
	}

	:global(.modal-content[data-scope='dialog'][data-part='content'][data-size='sm']) {
		--dialog-width: 26rem;
	}

	:global(.modal-content[data-scope='dialog'][data-part='content'][data-size='lg']) {
		--dialog-width: 48rem;

		height: min(42rem, calc(100dvh - 2 * var(--space-4)));
	}

	header:last-child,
	header:has(+ footer) {
		padding-bottom: var(--space-4);
	}

	header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-3) var(--space-2) var(--space-4);
	}

	header:focus {
		outline: none;
	}

	header > div {
		min-width: 0;
		padding-top: 6px;
	}

	header :global([data-scope='dialog'][data-part='title']) {
		font-size: var(--text-lg);
		line-height: 1.35;
		overflow-wrap: anywhere;
	}

	.body {
		flex: 1;
		min-height: 0;
		overflow-y: auto;
		padding: 0 var(--space-4) var(--space-4);
		overscroll-behavior: contain;
	}

	footer {
		display: flex;
		justify-content: flex-end;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-4);
		border-top: 1px solid var(--color-border);
		background: var(--color-surface-alt);
	}

	@media (max-width: 34rem) {
		:global(.modal-content[data-scope='dialog'][data-part='content'][data-size='lg']) {
			height: calc(100dvh - 2 * var(--space-2));
		}

		footer > :global(*) {
			flex: 1;
		}
	}
</style>
