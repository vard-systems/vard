<script lang="ts">
	import type { Application } from '#lib/api/index.js';
	import { Badge } from '#lib/components/ui/index.js';
	import { types } from './applications';

	let { application }: { application: Application } = $props();

	const kind = $derived(types[application.type]);
</script>

<div class="summary">
	<span class="mark"><i class={kind.icon} style:font-size="20px" aria-hidden="true"></i></span>
	<div class="text">
		<strong>{kind.label}</strong>
		<span>{kind.public ? 'Public client' : 'Confidential client'}</span>
	</div>
	<Badge tone={application.is_enabled ? 'success' : 'neutral'}>
		{application.is_enabled ? 'Enabled' : 'Disabled'}
	</Badge>
</div>

<style>
	.summary {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		margin-bottom: var(--space-5);
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
	}

	.mark {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		border-radius: var(--radius-md);
		background: var(--color-secondary-alt);
	}

	.text {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.text span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}
</style>
