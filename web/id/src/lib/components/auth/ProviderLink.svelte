<script lang="ts">
	import type { IconName } from '../ui/icon';

	type Props = {
		/** The server's own path, which redirects to the provider. */
		href: string;
		label: string;
		icon?: IconName;
		disabled?: boolean;
	};

	let { href, label, icon, disabled = false }: Props = $props();
</script>

<!-- A full navigation off this app, not a route of it. -->
<!-- eslint-disable svelte/no-navigation-without-resolve -->
<a
	class="provider"
	href={disabled ? undefined : href}
	aria-disabled={disabled}
	data-sveltekit-reload
>
	{#if icon}<i class={icon} aria-hidden="true"></i>{/if}
	<span>{label}</span>
</a>

<!-- eslint-enable svelte/no-navigation-without-resolve -->

<style>
	.provider {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
		min-height: 44px;
		padding: 0 var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		color: var(--color-text);
		font-size: var(--text-base);
		font-weight: 600;
		text-decoration: none;
	}

	.provider:hover {
		background: var(--color-surface-alt);
	}

	span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.provider[aria-disabled='true'] {
		opacity: 0.6;
		pointer-events: none;
	}
</style>
