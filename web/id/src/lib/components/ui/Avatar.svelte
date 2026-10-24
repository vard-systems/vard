<script lang="ts">
	import { initials } from '#lib/utils/format.js';

	type Props = {
		user: { first_name: string; last_name: string; email: string };
		size?: 'md' | 'lg';
	};

	let { user, size = 'md' }: Props = $props();
</script>

<span class={['avatar', size]} aria-hidden="true">{initials(user)}</span>

<style>
	.avatar {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		border-radius: 50%;
		background: var(--color-accent);
		color: var(--color-accent-text);
		font-weight: 700;
	}

	.md {
		width: 34px;
		height: 34px;
		font-size: var(--text-sm);
	}

	.lg {
		width: 64px;
		height: 64px;
		font-size: var(--text-xl);
	}
</style>
