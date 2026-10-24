<script module lang="ts">
	import type { IconName } from '../ui/index.js';

	/** The standard scopes, each with how the list draws it. An API's scope takes the API's own
	    words instead. */
	const standard: Record<string, IconName> = {
		openid: 'ri-user-line',
		profile: 'ri-profile-line',
		email: 'ri-mail-line',
		offline_access: 'ri-time-line',
		roles: 'ri-shield-user-line'
	};
</script>

<script lang="ts">
	import type { RequestedScope } from '#lib/api/index.js';
	import { useTranslator } from '#lib/i18n/index.js';

	type Props = { scopes: RequestedScope[] };

	let { scopes }: Props = $props();

	const t = useTranslator();

	function describe(scope: RequestedScope): string {
		if (scope.name in standard) return t(`consent.scope.${scope.name}`);
		return t('consent.scope.api', {
			api: scope.api ?? '',
			scope: scope.description || scope.name
		});
	}
</script>

<ul class="scopes">
	{#each scopes as scope (scope.name)}
		<li>
			<i class={standard[scope.name] ?? 'ri-key-2-line'} aria-hidden="true"></i>
			<span>{describe(scope)}</span>
		</li>
	{/each}
</ul>

<style>
	.scopes {
		display: grid;
		margin: 0;
		padding: 0;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		list-style: none;
	}

	li {
		display: grid;
		grid-template-columns: auto 1fr;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3);
	}

	li + li {
		border-top: 1px solid var(--color-border);
	}

	i {
		color: var(--color-text-hint);
	}
</style>
