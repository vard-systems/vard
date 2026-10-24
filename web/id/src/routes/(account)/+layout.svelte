<script lang="ts">
	import { AccountHeader } from '#lib/components/account/index.js';
	import { SecuredBy } from '#lib/components/layout/index.js';
	import type { LayoutProps } from './$types';

	let { data, children }: LayoutProps = $props();
</script>

<div class="frame">
	<AccountHeader user={data.user} languages={data.languages} language={data.language} />

	<main>
		{@render children()}
	</main>

	<footer>
		<SecuredBy />
	</footer>
</div>

<style>
	.frame {
		--account-width: 56rem;
		--account-gutter: var(--space-5);

		display: grid;
		grid-template-rows: auto 1fr auto;
		min-height: 100dvh;
	}

	main {
		width: 100%;
		max-width: var(--account-width);
		margin: 0 auto;
		padding: var(--space-6) var(--account-gutter) var(--space-7);
	}

	footer {
		display: flex;
		justify-content: center;
		padding: var(--space-5) 0;
	}

	@media (max-width: 40rem) {
		.frame {
			--account-gutter: var(--space-4);
		}
	}
</style>
