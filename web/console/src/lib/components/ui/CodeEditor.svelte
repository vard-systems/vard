<script lang="ts">
	import { Field as ArkField } from '@ark-ui/svelte/field';
	import Field from './Field.svelte';
	import { highlighted } from './highlight.svelte';

	type Props = {
		label: string;
		value: string;
		rows?: number;
		error?: string;
		readOnly?: boolean;
	};

	let { label, value = $bindable(''), rows = 12, error, readOnly }: Props = $props();

	/** The coloured copy of what is typed. A trailing newline gets a space
	    after it, or the copy would be a line shorter than the text area and
	    the two would scroll apart at the end. */
	const highlight = highlighted(() => (value.endsWith('\n') ? `${value} ` : value));

	let mirror: HTMLElement | undefined;

	function follow(event: Event) {
		const area = event.currentTarget as HTMLTextAreaElement;
		if (!mirror) return;
		mirror.scrollTop = area.scrollTop;
		mirror.scrollLeft = area.scrollLeft;
	}
</script>

<Field {label} {error} {readOnly} filled>
	<div class="code-editor" style:--rows={rows}>
		<div class="mirror" bind:this={mirror} aria-hidden="true">
			{#if highlight.html}
				<!-- Shiki's own output, from text it escaped itself. -->
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html highlight.html}
			{:else}
				<pre class="shiki"><code>{value}</code></pre>
			{/if}
		</div>

		<ArkField.Textarea
			bind:value
			readonly={readOnly}
			spellcheck={false}
			autocapitalize="off"
			autocomplete="off"
			wrap="off"
			onscroll={follow}
		/>
	</div>
</Field>

<style>
	.code-editor {
		--code-padding: var(--field-value-top) 14px 10px;

		position: relative;
		height: calc(var(--field-value-top) + var(--rows) * var(--field-line) + 10px);
	}

	.mirror,
	.code-editor :global([data-part='textarea']) {
		position: absolute;
		inset: 0;
	}

	.mirror {
		overflow: hidden;
		pointer-events: none;
	}

	.mirror :global(pre.shiki),
	:global(.field-box) .code-editor :global([data-part='textarea']) {
		margin: 0;
		padding: var(--code-padding);
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 400;
		line-height: var(--field-line);
		letter-spacing: normal;
		white-space: pre;
		tab-size: 2;
	}

	.mirror :global(pre.shiki) {
		min-height: 100%;
	}

	:global(.field-box) .code-editor :global([data-part='textarea']) {
		z-index: 1;
		height: 100%;
		min-height: 0;
		max-height: none;
		overflow: auto;
		resize: none;
		field-sizing: fixed;
		background: transparent;
		color: transparent;
		-webkit-text-fill-color: transparent;
		caret-color: var(--color-text);
	}

	:global(.field-box) .code-editor :global([data-part='textarea']::selection) {
		background: var(--selection);
	}
</style>
