<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import { Dialog } from '@ark-ui/svelte/dialog';
	import { Portal } from '@ark-ui/svelte/portal';
	import type { IconName } from '#lib/components/ui/index.js';
	import type { Admin } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import { IconButton, Kbd } from '#lib/components/ui/index.js';
	import { visibleGroups } from './sidebar/sections';

	type Command = {
		id: string;
		label: string;
		group: string;
		icon: IconName;
		href: string;
		external?: boolean;
	};

	let open = $state(false);
	let query = $state('');
	let active = $state(0);
	let list = $state<HTMLElement | null>(null);

	const admin = $derived(page.data.admin as Admin | undefined);

	const commands = $derived<Command[]>([
		...visibleGroups(admin).flatMap((group) =>
			group.items.map((item) => ({
				id: item.route,
				label: item.label,
				group: group.label,
				icon: item.icon,
				href: resolve(item.route)
			}))
		),
		{
			id: 'github',
			label: 'GitHub',
			group: 'Help',
			icon: 'ri-github-fill',
			href: BRAND.githubUrl,
			external: true
		}
	]);

	const matches = $derived(commands.filter((command) => fuzzy(command, query)));

	const grouped = $derived(
		matches.reduce<{ name: string; items: { command: Command; index: number }[] }[]>(
			(groups, command, index) => {
				const last = groups.at(-1);
				const item = { command, index };
				if (last?.name === command.group) last.items.push(item);
				else groups.push({ name: command.group, items: [item] });

				return groups;
			},
			[]
		)
	);

	const activeId = $derived(matches[active] ? `command-palette-option-${active}` : undefined);

	function fuzzy(command: Command, text: string): boolean {
		const needle = text.trim().toLowerCase();
		if (!needle) return true;

		const haystack = `${command.label} ${command.group}`.toLowerCase();
		let at = 0;

		for (const letter of needle) {
			if (letter === ' ') continue;

			at = haystack.indexOf(letter, at);
			if (at === -1) return false;
			at += 1;
		}

		return true;
	}

	function show() {
		query = '';
		active = 0;
		open = true;
	}

	/** Whether a key pressed here is somebody typing, which a shortcut must
	    leave alone: a "/" in a search box or a redirect URI is a character. */
	function typing(target: EventTarget | null): boolean {
		if (!(target instanceof HTMLElement)) return false;

		return (
			target.isContentEditable ||
			['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName) ||
			target.closest('[role="dialog"]') !== null
		);
	}

	function onKeydown(event: KeyboardEvent) {
		const slash =
			event.key === '/' &&
			!event.metaKey &&
			!event.ctrlKey &&
			!event.altKey &&
			!typing(event.target);
		const commandK = event.key.toLowerCase() === 'k' && (event.metaKey || event.ctrlKey);

		if (!slash && !commandK) return;

		event.preventDefault();

		if (open && commandK) open = false;
		else if (!open) show();
	}

	function onInputKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
			event.preventDefault();

			const step = event.key === 'ArrowDown' ? 1 : -1;
			const count = matches.length;
			if (count === 0) return;

			active = (active + step + count) % count;
			scrollIntoView();

			return;
		}

		if (event.key === 'Enter') {
			event.preventDefault();
			void run(matches[active]);
		}
	}

	function scrollIntoView() {
		requestAnimationFrame(() => {
			list?.querySelector('[data-active="true"]')?.scrollIntoView({ block: 'nearest' });
		});
	}

	async function run(command: Command | undefined) {
		if (!command) return;

		open = false;

		if (command.external) {
			window.open(command.href, '_blank', 'noopener,noreferrer');
			return;
		}

		// Already resolved, back where the list was built from the routes
		// themselves — by here it is an address, not a route to look up.
		// eslint-disable-next-line svelte/no-navigation-without-resolve
		await goto(command.href);
	}
</script>

<svelte:window onkeydown={onKeydown} />

<IconButton icon="ri-search-line" label="Search (press /)" size="sm" onclick={show} />

<Dialog.Root bind:open lazyMount unmountOnExit>
	<Portal>
		<Dialog.Backdrop class="palette-backdrop" />
		<Dialog.Positioner class="palette-positioner">
			<Dialog.Content class="palette-content" aria-label="Search…">
				<div class="field">
					<i class="ri-search-line" style:font-size="18px" aria-hidden="true"></i>
					<!-- svelte-ignore a11y_autofocus -->
					<input
						id="command-palette-search"
						name="command-palette-search"
						role="combobox"
						aria-label="Search pages"
						aria-controls="command-palette-results"
						aria-expanded={open}
						aria-haspopup="listbox"
						aria-autocomplete="list"
						aria-activedescendant={activeId}
						autofocus
						type="text"
						autocomplete="off"
						spellcheck="false"
						placeholder="Go to a page…"
						bind:value={query}
						oninput={() => (active = 0)}
						onkeydown={onInputKeydown}
					/>
				</div>

				<div
					id="command-palette-results"
					class="results"
					role="listbox"
					aria-label="Pages"
					bind:this={list}
				>
					{#each grouped as group, groupIndex (group.name)}
						<div class="group" role="group" aria-labelledby={`command-palette-group-${groupIndex}`}>
							<h2 id={`command-palette-group-${groupIndex}`}>{group.name}</h2>

							{#each group.items as item (item.command.id)}
								<button
									id={`command-palette-option-${item.index}`}
									type="button"
									class="row"
									role="option"
									aria-selected={item.index === active}
									tabindex="-1"
									data-active={item.index === active}
									onclick={() => run(item.command)}
									onmousemove={() => (active = item.index)}
								>
									<i class={item.command.icon} aria-hidden="true"></i>
									<span class="label">{item.command.label}</span>

									{#if item.command.external}
										<i class="ri-external-link-line" style:font-size="14px" aria-hidden="true"></i>
									{:else if item.index === active}
										<i class="ri-corner-down-left-line" style:font-size="14px" aria-hidden="true"
										></i>
									{/if}
								</button>
							{/each}
						</div>
					{:else}
						<p class="empty">{`Nothing matches “${query}”`}</p>
					{/each}
				</div>

				<footer>
					<span><Kbd>↑</Kbd><Kbd>↓</Kbd> move</span>
					<span><Kbd>↵</Kbd> open</span>
					<span><Kbd>esc</Kbd> close</span>
				</footer>
			</Dialog.Content>
		</Dialog.Positioner>
	</Portal>
</Dialog.Root>

<style>
	:global(.palette-positioner[data-scope='dialog'][data-part='positioner']) {
		align-items: flex-start;
		padding-top: 12vh;
	}

	:global(.palette-backdrop[data-scope='dialog'][data-part='backdrop']) {
		backdrop-filter: blur(2px);
	}

	:global(.palette-content[data-scope='dialog'][data-part='content']) {
		--dialog-width: 36rem;

		margin: 0 auto;
		max-height: min(26rem, 70vh);
	}

	.field {
		display: flex;
		flex-shrink: 0;
		align-items: center;
		gap: var(--space-2);
		padding: 0 var(--space-3);
		height: var(--control-height-lg);
		border-bottom: 1px solid var(--color-border);
		color: var(--color-text-hint);
	}

	.field input {
		flex: 1;
		min-width: 0;
		border: none;
		background: none;
		color: var(--color-text);
		font: inherit;
		font-size: var(--text-lg);
	}

	.field input:focus {
		outline: none;
	}

	.results {
		flex: 1;
		min-height: 0;
		padding: var(--space-1);
		overflow-y: auto;
	}

	.group h2 {
		margin: 0;
		padding: var(--space-2) var(--space-2) var(--space-1);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
		font-weight: 600;
		letter-spacing: 0.04em;
		text-transform: uppercase;
	}

	.row {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2);
		border: none;
		border-radius: var(--radius-sm);
		background: none;
		color: var(--color-text);
		font: inherit;
		font-size: var(--text-base);
		text-align: left;
		cursor: pointer;
	}

	.row[data-active='true'] {
		background: var(--color-secondary);
	}

	.row .label {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.row :global(i:last-child) {
		color: var(--color-text-hint);
	}

	.empty {
		margin: 0;
		padding: var(--space-5) var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-base);
		text-align: center;
	}

	footer {
		display: flex;
		flex-shrink: 0;
		gap: var(--space-3);
		padding: var(--space-2) var(--space-3);
		border-top: 1px solid var(--color-border);
		background: var(--color-surface-alt);
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	footer span {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	@media (max-width: 55rem) {
		footer {
			display: none;
		}
	}
</style>
