<script lang="ts">
	import { RiDatabase2Line, RiEraserLine, RiFilter3Line } from 'svelte-remixicon';
	import type { CacheGroup } from '#lib/api/index.js';
	import { Button, Icon, List, ListItem, Panel, Tag, Thumb } from '#lib/components/ui/index.js';

	type Props = {
		groups: CacheGroup[];
		current: string;
		clearing?: string;
		onFilter: (group: string) => void;
		onClear: (group: string) => void;
	};

	let { groups, current, clearing, onFilter, onClear }: Props = $props();

	function describe(group: CacheGroup): string {
		const values = `${group.entries} ${group.entries === 1 ? 'value' : 'values'}`;
		const stale = group.stale > 0 ? ` · ${group.stale} stale` : '';
		return `generation ${group.generation} · ${values}${stale}`;
	}
</script>

<Panel title="Groups" icon={RiDatabase2Line} flush>
	<List label="Cache groups">
		{#each groups as group (group.name)}
			<ListItem title={group.name} description={describe(group)}>
				{#snippet lead()}<Thumb icon={RiDatabase2Line} />{/snippet}
				{#snippet end()}
					<span class="end">
						{#if !group.known}
							<Tag small title="This server no longer reads it; what it holds will expire."
								>not read</Tag
							>
						{/if}
						{#if current === group.name}
							<Tag small tone="info" dot>shown below</Tag>
						{:else}
							<Button size="sm" variant="subtle" onclick={() => onFilter(group.name)}>
								<Icon icon={RiFilter3Line} />
								Show
							</Button>
						{/if}
						{#if group.known}
							<Button
								size="sm"
								variant="subtle"
								colorPalette="danger"
								loading={clearing === group.name}
								disabled={clearing !== undefined}
								onclick={() => onClear(group.name)}
							>
								<Icon icon={RiEraserLine} />
								Clear
							</Button>
						{/if}
					</span>
				{/snippet}
			</ListItem>
		{:else}
			<ListItem title="No groups" description="Nothing in this database is kept in groups." />
		{/each}
	</List>
</Panel>

<style>
	.end {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}
</style>
