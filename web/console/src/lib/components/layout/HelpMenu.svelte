<script lang="ts">
	import { RiBookOpenLine, RiGithubFill, RiKeyboardLine, RiQuestionLine } from 'svelte-remixicon';
	import { BRAND } from '#lib/brand.js';
	import {
		DropdownMenu,
		Icon,
		Kbd,
		MenuGroup,
		MenuInfo,
		MenuItem,
		MenuSeparator
	} from '#lib/components/ui/index.js';
</script>

<DropdownMenu label="Help and resources">
	{#snippet trigger()}<Icon icon={RiQuestionLine} size="1.125rem" />{/snippet}

	<MenuGroup label="Resources">
		<MenuItem
			value="documentation"
			icon={RiBookOpenLine}
			label="Documentation"
			description="Guides and the API reference"
			href={BRAND.docsUrl}
		/>
		<MenuItem
			value="github"
			icon={RiGithubFill}
			label="GitHub"
			description="Source, issues and releases"
			href={BRAND.githubUrl}
		/>
	</MenuGroup>

	<MenuSeparator />

	<MenuInfo icon={RiKeyboardLine} label="Search the panel">
		{#snippet end()}<Kbd>/</Kbd>{/snippet}
	</MenuInfo>
</DropdownMenu>
