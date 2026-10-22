<script lang="ts">
	import type { Language, LocaleApp } from '#lib/api/index.js';
	import { Badge, Code, type Column, DataTable, Tag, Tooltip } from '#lib/components/ui/index.js';

	type Props = {
		languages: Language[];
		apps: LocaleApp[];
		onOpen: (language: Language) => void;
		empty: string;
	};

	let { languages, apps, onOpen, empty }: Props = $props();

	type Row = Language & { id: string };

	const rows = $derived<Row[]>(languages.map((language) => ({ ...language, id: language.code })));

	const columns = $derived<Column[]>([
		{ key: 'language', label: 'Language', min: '14rem', grow: true },
		{ key: 'code', label: 'Code', min: '7rem' },
		{
			key: 'translated',
			label: 'Translated',
			min: '18rem'
		},
		{ key: 'status', label: 'Status', min: '11rem' }
	]);

	function appName(app: LocaleApp): string {
		return app === 'id' ? 'Sign-in pages' : app;
	}
</script>

<DataTable {columns} {rows} {empty} onOpen={(row) => onOpen(row)} label={(row) => row.name}>
	{#snippet row(language)}
		<td>
			<span class="names">
				<strong lang={language.code}>{language.native_name}</strong>
				<span class="english">{language.name}</span>
			</span>
		</td>

		<td><Code tone="quiet">{language.code}</Code></td>

		<td>
			<span class="bars">
				{#each apps as app (app)}
					{@const percent = language.coverage[app] ?? 0}
					{@const missing = language.missing[app] ?? 0}
					<Tooltip
						label={missing > 0
							? `${appName(app)} · ${missing} missing`
							: `${appName(app)} · Complete`}
					>
						{#snippet children(trigger)}
							<span class="bar" {...trigger()}>
								<span class="track">
									<span class="fill" class:whole={percent === 100} style="--filled: {percent}%"
									></span>
								</span>
								<span class="percent">{percent}%</span>
							</span>
						{/snippet}
					</Tooltip>
				{/each}
			</span>
		</td>

		<td>
			<span class="status">
				{#if language.is_default}
					<Tag tone="info" strong>default</Tag>
				{:else if language.is_enabled}
					<Badge tone="success">offered</Badge>
				{:else}
					<Badge>off</Badge>
				{/if}
			</span>
		</td>
	{/snippet}
</DataTable>

<style>
	.names {
		display: flex;
		flex-direction: column;
		min-width: 0;
		line-height: 1.3;
	}

	.english {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.bars {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.bar {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}

	.track {
		display: block;
		width: 4.5rem;
		height: 6px;
		border-radius: var(--radius-pill);
		background: var(--color-secondary-alt);
		overflow: hidden;
	}

	.fill {
		display: block;
		width: var(--filled);
		height: 100%;
		border-radius: inherit;
		background: var(--color-text-hint);
	}

	.fill.whole {
		background: var(--color-success);
	}

	.bar.none {
		justify-content: center;
		width: calc(4.5rem + 2.75rem);
		color: var(--color-text-hint);
	}

	.percent {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
		font-variant-numeric: tabular-nums;
	}

	.status {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
	}
</style>
