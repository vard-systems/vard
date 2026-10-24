<script lang="ts">
	import { createQuery } from '@tanstack/svelte-query';
	import type { UserRecord } from '#lib/api/index.js';
	import { IconButton, Note } from '#lib/components/ui/index.js';
	import { userSearchOptions } from '#lib/query/index.js';

	type Props = {
		user: UserRecord | null;
		/** Whether the administrator may see users at all. */
		allowed: boolean;
	};

	let { user = $bindable(null), allowed }: Props = $props();

	let search = $state('');

	const users = createQuery(() => ({
		...userSearchOptions(search.trim()),
		enabled: allowed && user === null
	}));
</script>

<div class="field">
	<span class="label">User</span>
	{#if !allowed}
		<Note>Your roles do not include seeing users.</Note>
	{:else if user}
		<div class="picked">
			<i class="ri-user-line" aria-hidden="true"></i>
			<span>{user.email}</span>
			<IconButton
				icon="ri-close-line"
				label="Choose another user"
				size="sm"
				onclick={() => (user = null)}
			/>
		</div>
	{:else}
		<div class="user-search">
			<label class="search">
				<i class="ri-search-line" aria-hidden="true"></i>
				<input
					id="token-preview-user-search"
					name="token-preview-user-search"
					type="search"
					placeholder="Search users by email or any field…"
					bind:value={search}
					onkeydown={(event) => event.key === 'Enter' && event.preventDefault()}
					aria-label="Search users"
				/>
			</label>
			<ul class="results">
				{#each users.data ?? [] as found (found.id)}
					<li>
						<button type="button" onclick={() => (user = found)}>
							<i class="ri-user-line" style:font-size="15px" aria-hidden="true"></i>
							{found.email}
						</button>
					</li>
				{:else}
					<li class="muted">{users.isPending ? 'Searching…' : 'No users match this.'}</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.label {
		font-size: var(--text-sm);
		font-weight: 600;
	}

	.muted {
		margin: 0;
		padding: var(--space-2);
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.picked {
		display: inline-flex;
		align-items: center;
		align-self: flex-start;
		gap: var(--space-2);
		padding: 2px 2px 2px var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-pill);
		font-size: var(--text-sm);
	}

	.user-search {
		overflow: hidden;
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
	}

	.search {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		height: 40px;
		padding: 0 12px;
		border-bottom: 1px solid var(--color-border);
		background: var(--color-input);
		color: var(--color-text-hint);
	}

	.search input {
		flex: 1;
		min-width: 0;
		border: none;
		background: transparent;
		color: var(--color-text);
		font: inherit;
	}

	.search input:focus {
		outline: none;
	}

	.results {
		margin: 0;
		padding: var(--space-1);
		list-style: none;
	}

	.results button {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2);
		border: none;
		border-radius: var(--radius-sm);
		background: transparent;
		color: var(--color-text);
		font: inherit;
		font-size: var(--text-sm);
		text-align: left;
		cursor: pointer;
	}

	.results button:hover {
		background: var(--row-hover);
	}
</style>
