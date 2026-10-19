<script lang="ts">
	import {
		RiDatabase2Line,
		RiFileList3Line,
		RiHardDrive2Line,
		RiShieldUserLine,
		RiSpeedUpLine,
		RiUserLine
	} from '#lib/icons.js';
	import type { CacheDatabase } from '#lib/api/index.js';
	import { StatCard } from '#lib/components/ui/index.js';

	type Props = { database: CacheDatabase };

	let { database }: Props = $props();

	const values = $derived(database.kinds.entry ?? 0);
	const stale = $derived(database.kinds.stale ?? 0);
	const megabytes = $derived(Math.round(database.memory_bytes / 1024 / 1024));
</script>

<div class="stats">
	<StatCard
		label="Keys"
		value={database.keys}
		icon={RiDatabase2Line}
		note={`database ${database.number}${database.version ? ` · Redis ${database.version}` : ''}`}
	/>

	{#if database.name === 'sessions'}
		<StatCard
			label="Administrators’ sessions"
			value={database.sessions.admin ?? 0}
			icon={RiShieldUserLine}
		/>
		<StatCard label="Users’ sessions" value={database.sessions.user ?? 0} icon={RiUserLine} />
		<StatCard
			label="Rate limits"
			value={database.kinds.ratelimit ?? 0}
			icon={RiSpeedUpLine}
			note="addresses being counted"
		/>
	{:else}
		<StatCard
			label="Cached values"
			value={values}
			icon={RiFileList3Line}
			note={stale > 0 ? `${stale} stale, left to expire` : 'none stale'}
		/>
		<StatCard
			label="Groups"
			value={database.groups.length}
			icon={RiDatabase2Line}
			note="each cleared as one"
		/>
		<StatCard
			label="Memory, MB"
			value={megabytes}
			icon={RiHardDrive2Line}
			note="the whole Redis server"
		/>
	{/if}
</div>

<style>
	.stats {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: var(--space-3);
	}

	@media (max-width: 80rem) {
		.stats {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}
	}

	@media (max-width: 34rem) {
		.stats {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
