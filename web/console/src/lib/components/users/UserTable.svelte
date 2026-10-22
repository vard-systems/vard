<script lang="ts">
	import type { Application, SocialKind, UserField, UserRecord } from '#lib/api/index.js';
	import { Badge, Code, type Column, DataTable, Tooltip } from '#lib/components/ui/index.js';
	import { markFor as providerMark } from '#lib/components/social/providers.js';
	import FieldValue from './FieldValue.svelte';
	import { valueOf } from './fields';

	type Props = {
		users: UserRecord[];
		fields: UserField[];
		applications: Application[];
		onOpen: (user: UserRecord) => void;
		selected?: string[];
		onSelect?: (ids: string[]) => void;
	};

	let { users, fields, applications, onOpen, selected, onSelect }: Props = $props();

	const appNames = $derived(new Map(applications.map((app) => [app.id, app.name])));

	const columns = $derived<Column[]>([
		{ key: 'id', label: 'ID', min: '10rem' },
		...fields.map((field) => ({
			key: field.name,
			label: field.label || field.name,
			min: field.type === 'email' ? '13rem' : field.type === 'bool' ? '8rem' : '9rem',
			grow: field.type === 'email'
		})),
		{ key: 'roles', label: 'Roles', min: '10rem' },
		{ key: 'signs_in_with', label: 'Signs in with', min: '9rem' }
	]);

	function markFor(kind: SocialKind) {
		return providerMark(kind);
	}

	function shortId(id: string): string {
		return id.replace(/-/g, '').slice(0, 15);
	}
</script>

<DataTable
	{columns}
	rows={users}
	empty="No users match this."
	{onOpen}
	label={(user) => `Edit ${user.email}`}
	{selected}
	{onSelect}
>
	{#snippet row(user)}
		<td><Code title={user.id}>{shortId(user.id)}</Code></td>

		{#each fields as field (field.name)}
			<td><FieldValue type={field.type} value={valueOf(user, field)} /></td>
		{/each}

		<td>
			<span class="roles">
				{#each user.roles ?? [] as role (role.id)}
					<Badge>
						{#if role.application_id}
							<span class="app">{appNames.get(role.application_id) ?? '?'}</span>
						{/if}
						{role.name}
					</Badge>
				{:else}
					<span class="empty">N/A</span>
				{/each}
			</span>
		</td>

		<td>
			<span class="ways">
				{#each user.social_accounts ?? [] as account (account.id)}
					{@const mark = markFor(account.kind)}
					<Tooltip label="{account.provider} · {account.email || 'no address'}">
						{#snippet children(trigger)}
							<span class="way" {...trigger()}>
								<i class={mark} aria-hidden="true"></i>
							</span>
						{/snippet}
					</Tooltip>
				{/each}

				{#if user.has_password}
					<Tooltip label="Signs in with a password">
						{#snippet children(trigger)}
							<span class="way" {...trigger()}
								><i class="ri-key-2-line" aria-hidden="true"></i></span
							>
						{/snippet}
					</Tooltip>
				{/if}

				{#if !user.has_password && (user.social_accounts ?? []).length === 0}
					<span class="empty">N/A</span>
				{/if}
			</span>
		</td>
	{/snippet}
</DataTable>

<style>
	.ways {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}

	.way {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		width: 26px;
		height: 26px;
		border: 1px solid var(--color-secondary-alt);
		border-radius: var(--radius-sm);
		color: var(--color-text-hint);
	}

	.roles {
		display: inline-flex;
		gap: var(--space-1);
	}

	.app {
		margin-right: 4px;
		opacity: 0.65;
	}

	.app::after {
		content: ' /';
	}

	.empty {
		color: var(--color-text-disabled);
		font-size: var(--text-sm);
	}
</style>
