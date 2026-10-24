<script module lang="ts">
	import type { IconName } from '../ui/icon';

	/** The mark drawn for each kind of provider the server knows (model.SocialSpecs). */
	const marks: Record<string, IconName | undefined> = {
		google: 'ri-google-fill',
		apple: 'ri-apple-fill',
		facebook: 'ri-facebook-circle-fill',
		vk: 'ri-vk-fill'
	};
</script>

<script lang="ts">
	import type { SocialProvider } from '#lib/api/index.js';
	import { socialHref } from '#lib/utils/links.js';
	import ProviderLink from './ProviderLink.svelte';

	type Props = {
		providers: SocialProvider[];
		/** The divider's words above the buttons. */
		label: string;
		request?: string | null;
		next?: string | null;
		disabled?: boolean;
	};

	let { providers, label, request = null, next = null, disabled = false }: Props = $props();
</script>

{#if providers.length > 0}
	<div class="social">
		<div class="divider"><span>{label}</span></div>

		<div class={['buttons', { single: providers.length === 1 }]}>
			{#each providers as provider (provider.slug)}
				<ProviderLink
					href={socialHref(provider.slug, { request, next })}
					label={provider.name}
					icon={marks[provider.kind]}
					{disabled}
				/>
			{/each}
		</div>
	</div>
{/if}

<style>
	.social {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		margin-top: var(--space-1);
	}

	.divider {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.divider::before,
	.divider::after {
		flex: 1;
		height: 1px;
		background: var(--color-border);
		content: '';
	}

	.buttons {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-2);
	}

	.buttons.single {
		grid-template-columns: 1fr;
	}
</style>
