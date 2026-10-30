<script lang="ts">
	import type { Snippet } from 'svelte';
	import { Badge, type IconName } from 'ui';

	type Props = {
		tone?: 'neutral' | 'success';
		icon?: IconName;
		children: Snippet;
	};

	let { tone = 'neutral', icon, children }: Props = $props();
</script>

<Badge {tone} {icon}>{@render children()}</Badge>
