<script lang="ts">
	/** The organization's terms and privacy policy, as the pages that ask for them list them. */
	type Props = { links: { label: string; href: string }[] };

	let { links }: Props = $props();
</script>

<ul class="links">
	{#each links as link (link.href)}
		<li>
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -- the organization's own page -->
			<a href={link.href} target="_blank" rel="noreferrer noopener">
				<i class="ri-file-text-line" aria-hidden="true"></i>
				{link.label}
				<i class="ri-external-link-line external" aria-hidden="true"></i>
			</a>
		</li>
	{/each}
</ul>

<style>
	.links {
		display: grid;
		gap: var(--space-2);
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.links a {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		color: var(--color-text);
		font-weight: 500;
		text-decoration: none;
	}

	.links a:hover {
		background: var(--color-surface-alt);
	}

	.external {
		margin-left: auto;
		color: var(--color-text-hint);
	}
</style>
