<script lang="ts">
	import { BRAND } from '$lib/brand';
	import { Icon, LanguagePicker, ThemeToggle } from '$lib/components';
	import { useTranslator } from '$lib/i18n';
	import { supportLinks } from '$lib/utils/legal';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();

	const t = useTranslator();

	const support = $derived(supportLinks(data.organization));
</script>

<div class="shell">
	<div class="corner">
		<LanguagePicker languages={data.languages} current={data.language} />
		<ThemeToggle />
	</div>

	<main>
		<div class="column">
			{@render children()}
		</div>
	</main>

	<footer>
		{#if support.length > 0}
			<p class="help">
				{t('shell.trouble')}
				{#each support as link, index (link.href)}
					{#if index > 0}<span aria-hidden="true">·</span>{/if}
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a href={link.href}>{link.label}</a>
				{/each}
			</p>
		{/if}

		<p class="secured">
			<Icon name="shield" size="0.9375rem" />
			<span>{t('shell.secured_by')} <strong>{BRAND.name}</strong></span>
		</p>
	</footer>
</div>

<style>
	.shell {
		display: grid;
		grid-template-rows: 1fr auto;
		min-height: calc(100dvh - (var(--app-inset) * 2));
		margin: var(--app-inset);
		border-radius: var(--app-radius);
		border: 1px solid var(--color-border);
		overflow: clip;
		padding: 0 var(--space-4);
		background:
			radial-gradient(
				60rem 28rem at 50% -8rem,
				color-mix(in srgb, var(--color-accent), transparent 88%),
				transparent
			),
			var(--color-body);
	}

	.corner {
		position: fixed;
		top: var(--space-3);
		right: var(--space-3);
		z-index: 10;
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	main {
		display: grid;
		place-items: center;
		padding: var(--space-7) 0 var(--space-5);
	}

	.column {
		width: 100%;
		max-width: 27rem;
	}

	footer {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-4) 0 var(--space-5);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		text-align: center;
	}

	.help {
		display: flex;
		align-items: center;
		justify-content: center;
		flex-wrap: wrap;
		gap: var(--space-1) var(--space-2);
	}

	.help a {
		color: var(--color-text);
		font-weight: 600;
	}

	.secured {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	@media (max-width: 30rem) {
		.shell {
			min-height: 100dvh;
			margin: 0;
			border-radius: 0;
			border: 0;
			background: var(--color-surface);
		}

		main {
			place-items: start center;
			padding-top: calc(var(--space-7) + var(--space-3));
		}
	}
</style>
