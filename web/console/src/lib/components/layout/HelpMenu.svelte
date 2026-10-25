<script lang="ts">
	import { BRAND } from '#lib/brand.js';
	import {
		DropdownMenu,
		Kbd,
		MenuGroup,
		MenuInfo,
		MenuItem,
		MenuSeparator
	} from '#lib/components/ui/index.js';
</script>

<DropdownMenu label="Help and resources">
	{#snippet trigger()}<i class="ri-question-line" style:font-size="18px" aria-hidden="true"
		></i>{/snippet}

	<MenuGroup label="Resources">
		{#each BRAND.links as link (link.id)}
			<MenuItem
				value={link.id}
				icon={link.icon}
				label={link.label}
				description={link.description}
				href={link.href}
			/>
		{/each}
	</MenuGroup>

	<MenuSeparator />

	<MenuInfo icon="ri-keyboard-line" label="Search the panel">
		{#snippet end()}<Kbd>/</Kbd>{/snippet}
	</MenuInfo>
</DropdownMenu>
