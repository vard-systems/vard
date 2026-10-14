<script lang="ts">
	import type { Attachment } from 'svelte/attachments';
	import { RiArrowLeftLine, RiArrowRightLine, RiEditLine } from 'svelte-remixicon';
	import { BRAND, LANGUAGE_KEY } from '#lib/brand.js';
	import { href } from '#lib/docs/links.js';
	import Icon from '#lib/components/Icon.svelte';
	import Toc from '#lib/components/Toc.svelte';

	let { data } = $props();

	const home = $derived(data.file.endsWith('content/index.md'));
	const editURL = $derived(`${BRAND.githubURL}/edit/main/web/docs/${data.file}`);

	/** Every block of examples on the page shows one language: the reader's. */
	function showLanguage(article: HTMLElement, language: string) {
		for (const input of article.querySelectorAll<HTMLInputElement>('input[data-language]')) {
			if (input.value === language) input.checked = true;
		}
	}

	// The rendered Markdown carries the copy buttons and the language tabs, so
	// one listener on the article answers all of them.
	const behaviour: Attachment<HTMLElement> = (article) => {
		try {
			const saved = localStorage.getItem(LANGUAGE_KEY);
			if (saved) showLanguage(article, saved);
		} catch {
			// Nothing saved in a private window: the first tab it is.
		}

		function onchange(event: Event) {
			const input = event.target as HTMLInputElement;
			if (!input.matches('input[data-language]')) return;
			showLanguage(article, input.value);
			try {
				localStorage.setItem(LANGUAGE_KEY, input.value);
			} catch {
				// The choice lasts for this page.
			}
		}

		async function onclick(event: MouseEvent) {
			const button = (event.target as HTMLElement).closest<HTMLButtonElement>('[data-copy]');
			if (!button) return;

			// In a tabbed block, the one panel showing.
			const block = button.closest('.code-block');
			const code = [...(block?.querySelectorAll('pre') ?? [])].find((pre) => pre.offsetParent);
			const label = button.querySelector('span');
			if (!code || !label) return;

			try {
				await navigator.clipboard.writeText(code.textContent ?? '');
				label.textContent = 'Copied';
				button.dataset.copied = '';
			} catch {
				label.textContent = 'Select and copy';
			}
			setTimeout(() => {
				label.textContent = 'Copy';
				delete button.dataset.copied;
			}, 1600);
		}

		article.addEventListener('change', onchange);
		article.addEventListener('click', onclick);
		return () => {
			article.removeEventListener('change', onchange);
			article.removeEventListener('click', onclick);
		};
	};
</script>

<svelte:head>
	<title>{home ? `${BRAND.name} documentation` : `${data.title} · ${BRAND.name} Docs`}</title>
	{#if data.description}
		<meta name="description" content={data.description} />
	{/if}
</svelte:head>

<div class="page">
	<article>
		{#if !home}
			<header>
				<p class="eyebrow">{data.group}</p>
				<h1>{data.title}</h1>
				{#if data.description}
					<p class="lead">{data.description}</p>
				{/if}
			</header>
		{/if}

		<!-- eslint-disable-next-line svelte/no-at-html-tags -- our own Markdown, rendered while building -->
		<div class="prose" {@attach behaviour}>{@html data.html}</div>

		<footer>
			<!-- eslint-disable svelte/no-navigation-without-resolve -- href() is resolve(), and the editor is another site -->
			<nav class="pager" aria-label="Previous and next page">
				{#if data.previous}
					<a class="previous" href={href(data.previous.href)}>
						<small><Icon icon={RiArrowLeftLine} size="0.875rem" /> Previous</small>
						{data.previous.title}
					</a>
				{/if}
				{#if data.next}
					<a class="next" href={href(data.next.href)}>
						<small>Next <Icon icon={RiArrowRightLine} size="0.875rem" /></small>
						{data.next.title}
					</a>
				{/if}
			</nav>

			<p class="provenance">
				{#if data.generated}
					Written from the server's code by <code>make docs</code>. To change this page, change the
					code.
				{:else}
					<a href={editURL} target="_blank" rel="noopener noreferrer">
						<Icon icon={RiEditLine} size="0.875rem" /> Edit this page
					</a>
				{/if}
			</p>
			<!-- eslint-enable svelte/no-navigation-without-resolve -->
		</footer>
	</article>

	<aside class="toc">
		<Toc entries={data.toc} />
	</aside>
</div>

<style>
	.page {
		display: grid;
		grid-template-columns: minmax(0, var(--content-width)) var(--toc-width);
		justify-content: center;
		gap: var(--space-6);
		padding: var(--space-5) var(--page-gutter) var(--space-6);
	}

	header {
		margin-bottom: var(--space-4);
	}

	.eyebrow {
		margin: 0 0 var(--space-1);
		color: var(--color-info);
		font-size: var(--text-xs);
		font-weight: 600;
	}

	h1 {
		margin: 0;
		font-size: var(--text-2xl);
		font-weight: 700;
		letter-spacing: -0.015em;
		line-height: 1.25;
	}

	.lead {
		margin: var(--space-1) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-lg);
	}

	footer {
		margin-top: var(--space-6);
	}

	.pager {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--space-2);
	}

	.pager a {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		color: var(--color-text);
		font-weight: 500;
		text-decoration: none;
		transition: border-color var(--speed-fast);
	}

	.pager a:hover {
		border-color: var(--color-brand);
	}

	.pager small {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		color: var(--color-text-hint);
		font-size: var(--text-xs);
		font-weight: 400;
	}

	.next {
		grid-column: 2;
		align-items: flex-end;
		text-align: right;
	}

	.provenance {
		margin: var(--space-4) 0 0;
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.provenance a {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		color: var(--color-text-hint);
		text-decoration: none;
	}

	.provenance a:hover {
		color: var(--color-text);
	}

	.provenance code {
		font-size: 11.5px;
	}

	@media (max-width: 75rem) {
		.page {
			grid-template-columns: minmax(0, var(--content-width));
		}

		.toc {
			display: none;
		}
	}

	@media (max-width: 34rem) {
		.pager {
			grid-template-columns: 1fr;
		}

		.next {
			grid-column: 1;
		}
	}
</style>
