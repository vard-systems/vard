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
		<MenuItem
			value="github"
			icon="ri-github-fill"
			label="GitHub"
			description="Source, issues and releases"
			href={BRAND.githubUrl}
		/>
	</MenuGroup>

	<MenuSeparator />

	<MenuInfo icon="ri-keyboard-line" label="Search the panel">
		{#snippet end()}<Kbd>/</Kbd>{/snippet}
	</MenuInfo>
</DropdownMenu>
