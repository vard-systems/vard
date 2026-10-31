<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { Application, Organization } from '#lib/api/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import { legalLinks } from '#lib/utils/legal.js';
	import AppMark from './AppMark.svelte';

	type Props = {
		application?: Application | null;
		organization?: Organization | null;
		title: string;
		subtitle?: string;
		children: Snippet;
		below?: Snippet;
	};

	let {
		application = null,
		organization = null,
		title,
		subtitle,
		children,
		below
	}: Props = $props();

	const t = useTranslator();
	const uid = $props.id();

	const legal = $derived(legalLinks(application, organization));
</script>

<section class="card" aria-labelledby="auth-title-{uid}">
	<header>
		<AppMark
			name={application?.name ?? organization?.name}
			logo={application?.logo_url || organization?.logo_url}
		/>
		<h1 id="auth-title-{uid}">{title}</h1>
		{#if subtitle}<p class="subtitle">{subtitle}</p>{/if}
	</header>

	<div class="body">
		{@render children()}
	</div>
</section>

{#if below}
	<p class="below">{@render below()}</p>
{/if}

{#if legal.length > 0}
	<nav class="legal" aria-label={t('legal.label')}>
		{#each legal as link (link.key)}
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={link.href} target="_blank" rel="noopener noreferrer">{t(link.key)}</a>
		{/each}
	</nav>
{/if}

<style>
	.card {
		width: 100%;
		padding: var(--space-6);
		border: 1px solid var(--color-border);
		border-radius: 18px;
		background: var(--color-surface);
		box-shadow: var(--shadow-card);
	}

	header {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-5);
		text-align: center;
	}

	h1 {
		margin-top: var(--space-2);
		font-size: 1.5rem;
		font-weight: 700;
		line-height: 1.25;
		letter-spacing: -0.01em;
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.subtitle {
		color: var(--color-text-hint);
		font-size: var(--text-lg);
	}

	.below {
		margin-top: var(--space-5);
		text-align: center;
		color: var(--color-text-hint);
	}

	.below :global(a) {
		font-weight: 600;
	}

	.legal {
		display: flex;
		justify-content: center;
		flex-wrap: wrap;
		gap: var(--space-1) var(--space-5);
		margin-top: var(--space-3);
		font-size: var(--text-sm);
	}

	.legal a {
		color: var(--color-text-hint);
	}

	.legal a:hover {
		color: var(--color-text);
	}

	@media (max-width: 30rem) {
		.card {
			padding: var(--space-2) 0 0;
			border: 0;
			border-radius: 0;
			background: transparent;
			box-shadow: none;
		}
	}
</style>
