<script lang="ts">
	import {
		RiAppsLine,
		RiCheckLine,
		RiKey2Line,
		RiLockLine,
		RiShieldUserLine,
		RiTimerLine
	} from 'svelte-remixicon';
	import type { API } from '#lib/api/index.js';
	import { CodeBlock, FormSection, Icon, Input } from '#lib/components/ui/index.js';
	import { lifetimeLabel } from './scopes';

	type Props = {
		api: API;
		onTab: (tab: string) => void;
	};

	let { api, onTab }: Props = $props();

	const defaults = $derived(api.scopes.filter((scope) => scope.is_default).length);

	const stats = $derived([
		{
			icon: RiLockLine,
			label: 'Scopes',
			value: String(api.scopes.length),
			detail: defaults > 0 ? `${defaults} default` : 'None default',
			tab: 'scopes'
		},
		{
			icon: RiAppsLine,
			label: 'Applications',
			value: String(api.application_count),
			detail: 'Authorised to request tokens',
			tab: 'applications'
		},
		{
			icon: RiShieldUserLine,
			label: 'Roles',
			value: String(api.role_count),
			detail: api.enforce_roles ? 'Granting its scopes' : 'Not checked: role-based access off',
			tab: 'settings'
		},
		{
			icon: RiTimerLine,
			label: 'Access tokens',
			value: lifetimeLabel(api.token_lifetime),
			detail: `${api.signing_algorithm} · refresh tokens ${api.allow_offline_access ? 'allowed' : 'off'}`,
			tab: 'settings'
		}
	]);

	const example = $derived(
		JSON.stringify(
			{
				header: { alg: api.signing_algorithm, typ: 'at+jwt', kid: '…' },
				payload: {
					iss: api.issuer,
					aud: api.identifier,
					sub: '<user id>',
					client_id: '<application client id>',
					scope: api.scopes
						.slice(0, 2)
						.map((scope) => scope.name)
						.join(' '),
					iat: 1767225600,
					exp: 1767225600 + (api.token_lifetime || 3600)
				}
			},
			null,
			2
		)
	);
</script>

<div class="stats">
	{#each stats as stat (stat.label)}
		<button type="button" class="stat" onclick={() => onTab(stat.tab)}>
			<span class="stat-label"><Icon icon={stat.icon} size="0.875rem" />{stat.label}</span>
			<strong>{stat.value}</strong>
			<small>{stat.detail}</small>
		</button>
	{/each}
</div>

<div class="columns">
	<div>
		<FormSection
			title="Connection details"
			description="What the API's own code needs to validate the access tokens it receives."
		>
			<Input
				label="Audience (identifier)"
				value={api.identifier}
				readOnly
				copyable
				hint="Compare the token's aud claim with this."
			/>
			<Input
				label="Issuer"
				value={api.issuer}
				readOnly
				copyable
				hint="Compare the token's iss claim with this."
			/>
			<Input
				label="JWKS URI"
				icon={RiKey2Line}
				value={api.jwks_uri}
				readOnly
				copyable
				hint="The public keys tokens are signed with. Published once token issuing is enabled."
			/>
		</FormSection>
	</div>

	<div>
		<FormSection
			title="Validating a token"
			description="Every request to the API should pass all of these before anything else runs."
		>
			<ol class="checks">
				<li>
					<Icon icon={RiCheckLine} size="0.875rem" />
					<span>
						The signature verifies with a key from the JWKS URI, and <code>alg</code> is
						<code>{api.signing_algorithm}</code>. Never accept <code>none</code> or HS256.
					</span>
				</li>
				<li>
					<Icon icon={RiCheckLine} size="0.875rem" />
					<span><code>typ</code> is <code>at+jwt</code>, so an ID token is refused.</span>
				</li>
				<li>
					<Icon icon={RiCheckLine} size="0.875rem" />
					<span><code>iss</code> is the issuer and <code>aud</code> contains the identifier.</span>
				</li>
				<li>
					<Icon icon={RiCheckLine} size="0.875rem" />
					<span><code>exp</code> has not passed, allowing a minute of clock skew at most.</span>
				</li>
				<li>
					<Icon icon={RiCheckLine} size="0.875rem" />
					<span><code>scope</code> contains the scope the endpoint needs.</span>
				</li>
			</ol>

			<p class="important">
				<strong>Important:</strong> the API must still enforce permissions. This server decides which
				scopes a token carries; only the API can check that the scope an endpoint needs is there, and
				that the user may act on the particular record.
			</p>
		</FormSection>
	</div>
</div>

<FormSection
	title="Example access token"
	description="A decoded token for this API. The scope claim holds what the application is allowed and, with role-based access, what the user's roles grant."
>
	<CodeBlock code={example} copy="example token" />
</FormSection>

<style>
	.stats {
		display: grid;
		grid-template-columns: repeat(4, minmax(0, 1fr));
		gap: var(--space-3);
		margin-bottom: var(--space-5);
	}

	.stat {
		display: flex;
		flex-direction: column;
		gap: 4px;
		min-width: 0;
		padding: var(--space-3) var(--space-4);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-surface);
		background: var(--color-surface);
		color: var(--color-text);
		font: inherit;
		text-align: left;
		cursor: pointer;
		transition: border-color var(--speed-fast);
	}

	.stat:hover {
		border-color: var(--color-text-disabled);
	}

	.stat-label {
		display: flex;
		align-items: center;
		gap: 6px;
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.stat strong {
		font-size: var(--text-xl);
		font-weight: 600;
	}

	.stat small {
		color: var(--color-text-hint);
		font-size: var(--text-xs);
	}

	.columns {
		display: grid;
		grid-template-columns: repeat(2, minmax(0, 1fr));
		gap: var(--space-5) var(--space-6);
		margin-bottom: var(--space-5);
		padding-bottom: var(--space-5);
		border-bottom: 1px solid var(--color-border);
	}

	.checks {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
		margin: 0;
		padding: 0;
		list-style: none;
		font-size: var(--text-sm);
	}

	.checks li {
		display: flex;
		gap: var(--space-2);
		line-height: 1.5;
	}

	.checks li :global(svg) {
		flex-shrink: 0;
		margin-top: 3px;
		color: var(--color-success);
	}

	code {
		font-family: var(--font-mono);
		font-size: 0.92em;
	}

	.important {
		margin: 0;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius-sm);
		background: var(--surface-info);
		font-size: var(--text-sm);
		line-height: 1.5;
	}

	@media (max-width: 64rem) {
		.stats {
			grid-template-columns: repeat(2, minmax(0, 1fr));
		}

		.columns {
			grid-template-columns: minmax(0, 1fr);
		}
	}
</style>
