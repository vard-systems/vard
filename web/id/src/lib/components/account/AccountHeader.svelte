<script module lang="ts">
	import type { IconName } from '../ui/icon';

	type Section = { path: '' | 'security' | 'applications'; key: string; icon: IconName };

	const sections: Section[] = [
		{ path: '', key: 'account.profile', icon: 'ri-user-line' },
		{ path: 'security', key: 'account.security', icon: 'ri-lock-line' },
		{ path: 'applications', key: 'account.applications', icon: 'ri-apps-line' }
	];
</script>

<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { signIn, type PublicLanguage, type User } from '#lib/api/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import { displayName } from '#lib/utils/format.js';
	import { Brand, LanguagePicker, ThemeToggle } from '../layout/index.js';
	import { Avatar } from '../ui/index.js';

	type Props = {
		user: User;
		languages: PublicLanguage[];
		language: string;
	};

	let { user, languages, language }: Props = $props();

	const t = useTranslator();

	let signingOut = $state(false);

	async function signOut() {
		signingOut = true;
		try {
			await signIn.logout();
		} finally {
			// One navigation that reloads the data: invalidating first would
			// make the account layout's load redirect to sign in, racing this goto.
			await goto(resolve('login'), { refreshAll: true, replace: true });
		}
	}
</script>

<header>
	<div class="bar">
		<a class="home" href={resolve('')} aria-label={t('account.your_account')}><Brand /></a>

		<div class="end">
			<LanguagePicker {languages} current={language} />
			<ThemeToggle />
			<div class="who">
				<Avatar {user} />
				<span class="text">
					<strong>{displayName(user)}</strong>
					<span>{user.email}</span>
				</span>
			</div>
			<button class="signout" type="button" onclick={signOut} disabled={signingOut}>
				<i class="ri-logout-box-r-line" aria-hidden="true"></i>
				<span>{t('action.sign_out')}</span>
			</button>
		</div>
	</div>

	<nav aria-label={t('account.nav')}>
		{#each sections as section (section.path)}
			<a
				href={resolve(section.path)}
				aria-current={page.url.pathname === resolve(section.path) ? 'page' : undefined}
			>
				<i class={section.icon} style:font-size="16px" aria-hidden="true"></i>
				{t(section.key)}
			</a>
		{/each}
	</nav>
</header>

<style>
	header {
		position: sticky;
		top: 0;
		z-index: 10;
		border-bottom: 1px solid var(--color-border);
		background: color-mix(in srgb, var(--color-surface), transparent 8%);
		backdrop-filter: blur(10px);
		overscroll-behavior: none;
	}

	.bar,
	nav {
		width: 100%;
		max-width: var(--account-width);
		margin: 0 auto;
		padding-left: var(--account-gutter);
		padding-right: var(--account-gutter);
	}

	.bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-3);
		height: 64px;
	}

	.home:hover {
		text-decoration: none;
	}

	.end {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
	}

	.who {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		min-width: 0;
		padding: 0 var(--space-2);
	}

	.text {
		display: flex;
		flex-direction: column;
		min-width: 0;
		line-height: 1.25;
	}

	.text strong,
	.text span {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.text span {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	.signout {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		height: 38px;
		padding: 0 var(--space-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		background: var(--color-surface);
		font-weight: 600;
		cursor: pointer;
	}

	.signout:hover {
		background: var(--color-input);
	}

	nav {
		display: flex;
		gap: var(--space-1);
		overflow-x: auto;
		overscroll-behavior: none;
		scrollbar-width: none;
	}

	nav a {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-3);
		border-bottom: 2px solid transparent;
		color: var(--color-text-hint);
		font-weight: 600;
		white-space: nowrap;
	}

	nav a:hover {
		color: var(--color-text);
		text-decoration: none;
	}

	nav a[aria-current='page'] {
		border-bottom-color: var(--color-primary);
		color: var(--color-text);
	}

	@media (max-width: 40rem) {
		.text,
		.signout span {
			display: none;
		}
	}
</style>
