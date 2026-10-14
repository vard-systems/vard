<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Overview } from '#lib/api/index.js';
	import { countOf, formatShortDay } from '#lib/utils/format.js';

	type Props = {
		daily: Overview['daily'];
		href?: (day: string) => string;
	};

	let { daily, href }: Props = $props();

	function open() {
		if (hovered === null || !href) return;
		// The caller resolved the address.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		void goto(href(daily[hovered].day));
	}

	/** The plot's size in pixels, so the lines stay 2px whatever the width. */
	let width = $state(0);
	const height = 190;
	const pad = { top: 12, right: 10, bottom: 30, left: 34 };

	const plotWidth = $derived(Math.max(0, width - pad.left - pad.right));
	const plotHeight = height - pad.top - pad.bottom;

	const ceiling = $derived.by(() => {
		const most = Math.max(0, ...daily.map((day) => day.events));
		if (most <= 4) return 4;

		const padded = most * 1.2;
		const step = 10 ** Math.floor(Math.log10(padded));
		const rounded = Math.ceil(padded / step) * step;

		return rounded % 2 === 0 ? rounded : rounded + step;
	});

	const step = $derived(daily.length > 0 ? plotWidth / daily.length : 0);
	const y = (value: number) => pad.top + plotHeight - (value / ceiling) * plotHeight;

	function steppedPath(values: number[]): string {
		if (values.length === 0 || step === 0) return '';

		let d = `M ${pad.left} ${y(values[0])}`;
		values.forEach((value, index) => {
			d += ` V ${y(value)} H ${pad.left + step * (index + 1)}`;
		});

		return d;
	}

	function area(values: number[]): string {
		const line = steppedPath(values);
		if (!line) return '';

		const baseline = pad.top + plotHeight;
		return `${line} V ${baseline} H ${pad.left} Z`;
	}

	const events = $derived(daily.map((day) => day.events));
	const failures = $derived(daily.map((day) => day.failures));
	const anyFailures = $derived(failures.some((value) => value > 0));

	const ticks = $derived([0, ceiling / 2, ceiling]);

	const labelled = $derived.by(() => {
		const room = Math.max(1, Math.floor(plotWidth / 64));
		const every = Math.max(1, Math.ceil(daily.length / room));

		return new Set(
			daily.map((_, index) => index).filter((index) => (daily.length - 1 - index) % every === 0)
		);
	});

	let hovered = $state<number | null>(null);

	function track(event: PointerEvent) {
		const box = (event.currentTarget as SVGElement).getBoundingClientRect();
		const x = event.clientX - box.left - pad.left;

		hovered = x < 0 || x > plotWidth ? null : Math.min(daily.length - 1, Math.floor(x / step));
	}

	const tip = $derived.by(() => {
		if (hovered === null) return null;

		const day = daily[hovered];
		const left = pad.left + step * hovered + step / 2;

		return {
			day,
			left: Math.min(Math.max(left, 70), width - 70),
			top: y(day.events)
		};
	});
</script>

<div class="chart" bind:clientWidth={width}>
	{#if width > 0}
		<svg
			{width}
			{height}
			role="presentation"
			class:clickable={href !== undefined}
			onpointermove={track}
			onpointerleave={() => (hovered = null)}
			onclick={open}
		>
			{#each ticks as tick (tick)}
				<line class="grid" x1={pad.left} x2={width - pad.right} y1={y(tick)} y2={y(tick)} />
				<text
					class="axis"
					x={pad.left - 8}
					y={y(tick)}
					text-anchor="end"
					dominant-baseline="middle"
				>
					{tick}
				</text>
			{/each}

			{#if hovered !== null}
				<rect
					class="focus"
					x={pad.left + step * hovered}
					y={pad.top}
					width={step}
					height={plotHeight}
				/>
			{/if}

			<path class="events-fill" d={area(events)} />
			<path class="events-line" d={steppedPath(events)} />

			{#if anyFailures}
				<path class="failures-fill" d={area(failures)} />
				<path class="failures-line" d={steppedPath(failures)} />
			{/if}

			{#each daily as day, index (day.day)}
				{#if labelled.has(index)}
					<line
						class="grid"
						x1={pad.left + step * index + step / 2}
						x2={pad.left + step * index + step / 2}
						y1={pad.top + plotHeight}
						y2={pad.top + plotHeight + 5}
					/>
					<text
						class="axis"
						x={pad.left + step * index + step / 2}
						y={height - 8}
						text-anchor="middle"
					>
						{index === daily.length - 1 ? 'Today' : formatShortDay(day.day)}
					</text>
				{/if}
			{/each}
		</svg>

		{#if tip}
			<div class="tooltip" style:left="{tip.left}px" style:top="{tip.top}px">
				<div class="primary">
					{tip.day.events.toLocaleString()}
					{tip.day.events === 1 ? 'event' : 'events'}
				</div>
				<div class="secondary">
					{formatShortDay(tip.day.day)}{#if tip.day.failures > 0}
						· {tip.day.failures} refused{/if}
				</div>
				{#if href}<div class="secondary">Click to see the entries</div>{/if}
			</div>
		{/if}
	{/if}

	<table class="sr-only">
		<caption>Events per day</caption>
		<tbody>
			{#each daily as day (day.day)}
				<tr>
					<th scope="row">
						{#if href}
							<!-- The caller resolved the address. -->
							<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
							<a href={href(day.day)}>{formatShortDay(day.day)}</a>
						{:else}
							{formatShortDay(day.day)}
						{/if}
					</th>
					<td>{countOf(day.events, 'event')}</td>
					<td>{countOf(day.failures, 'refused sign-in')}</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>

<style>
	.clickable {
		cursor: pointer;
	}

	.chart {
		position: relative;
		display: flex;
		align-items: center;
		width: 100%;
		height: 100%;
		min-height: 190px;
	}

	svg {
		display: block;
		overflow: visible;
	}

	.grid {
		stroke: var(--color-secondary);
		stroke-width: 1;
	}

	.axis {
		fill: var(--color-text-hint);
		font-size: 11px;
		font-variant-numeric: tabular-nums;
	}

	.events-fill {
		fill: var(--color-secondary);
	}

	.events-line {
		fill: none;
		stroke: var(--color-border);
		stroke-width: 2;
	}

	.failures-fill {
		fill: color-mix(in srgb, var(--color-danger), transparent 85%);
	}

	.failures-line {
		fill: none;
		stroke: var(--color-danger);
		stroke-width: 1.5;
	}

	.focus {
		fill: var(--color-surface-alt);
	}

	.tooltip {
		position: absolute;
		z-index: 3;
		padding: 5px;
		border-radius: var(--radius-sm);
		background: var(--tooltip-surface);
		box-shadow: var(--shadow-panel);
		color: var(--tooltip-text);
		font-size: var(--text-sm);
		line-height: 1;
		text-align: center;
		white-space: nowrap;
		pointer-events: none;
		transform: translate(-50%, calc(-100% - 8px));
	}

	.primary {
		font-weight: bold;
	}

	.secondary {
		margin: 3px 0 1px;
		font-size: 0.85em;
		opacity: 0.6;
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip-path: inset(50%);
		white-space: nowrap;
	}
</style>
