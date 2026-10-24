<script lang="ts">
	import type { User } from '#lib/api/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import { Avatar, Badge } from '../ui/index.js';

	let { user }: { user: User } = $props();

	const t = useTranslator();

	const name = $derived(`${user.first_name} ${user.last_name}`.trim());
</script>

<div class="intro">
	<Avatar {user} size="lg" />
	<div>
		<h1>{name || t('account.your_account')}</h1>
		<p>
			{user.email}
			{#if user.email_verified}
				<Badge tone="success" icon="ri-check-line">{t('profile.verified')}</Badge>
			{:else}
				<Badge>{t('profile.not_verified')}</Badge>
			{/if}
		</p>
	</div>
</div>

<style>
	.intro {
		display: flex;
		align-items: center;
		gap: var(--space-4);
		margin-bottom: var(--space-2);
	}

	h1 {
		font-size: var(--text-2xl);
		line-height: 1.2;
	}

	p {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: var(--space-2);
		margin-top: var(--space-1);
		color: var(--color-text-hint);
		overflow-wrap: anywhere;
	}
</style>
