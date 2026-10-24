<script lang="ts">
	type Props = {
		name?: string;
		logo?: string;
		size?: number;
	};

	let { name = '', logo = '', size = 56 }: Props = $props();

	let failed = $state(false);

	const initial = $derived(name.trim().charAt(0).toUpperCase());
</script>

{#if logo && !failed}
	<img
		class="mark"
		src={logo}
		alt="{name} logo"
		style="--size: {size}px"
		onerror={() => (failed = true)}
	/>
{:else}
	<span class="mark initial" style="--size: {size}px" aria-hidden="true">
		{#if initial}{initial}{:else}<i
				class="ri-shield-keyhole-line"
				style:font-size="{size * 0.45}px"
				aria-hidden="true"
			></i>{/if}
	</span>
{/if}

<style>
	.mark {
		display: grid;
		place-items: center;
		flex-shrink: 0;
		width: var(--size);
		height: var(--size);
		border: 1px solid var(--color-border);
		border-radius: calc(var(--size) * 0.25);
		background: var(--color-surface);
		object-fit: contain;
	}

	.initial {
		border: 0;
		background: var(--color-accent);
		color: var(--color-accent-text);
		font-size: calc(var(--size) * 0.42);
		font-weight: 700;
	}
</style>
