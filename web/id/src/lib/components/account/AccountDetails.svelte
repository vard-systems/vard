<script lang="ts">
	import type { User } from '#lib/api/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import { formatDate } from '#lib/utils/format.js';
	import { Panel } from '../ui/index.js';

	type Props = {
		user: User;
		/** The organisation's, in which every date on these pages is shown. */
		timezone: string;
	};

	let { user, timezone }: Props = $props();

	const t = useTranslator();
</script>

<Panel title={t('profile.details_title')}>
	<dl>
		<div>
			<dt>{t('profile.member_since')}</dt>
			<dd>{formatDate(user.created_at, t, timezone)}</dd>
		</div>
		<div>
			<dt>{t('profile.last_sign_in')}</dt>
			<dd>
				{user.last_login_at ? formatDate(user.last_login_at, t, timezone) : t('time.never')}
			</dd>
		</div>
		<div>
			<dt>{t('profile.account_id')}</dt>
			<dd><code>{user.id}</code></dd>
		</div>
	</dl>
</Panel>

<style>
	dl {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		gap: var(--space-4);
		margin: 0;
	}

	dt {
		color: var(--color-text-hint);
		font-size: var(--text-sm);
	}

	dd {
		margin: var(--space-1) 0 0;
		font-weight: 600;
		overflow-wrap: anywhere;
	}

	code {
		font-family: var(--font-mono);
		font-size: var(--text-sm);
		font-weight: 400;
	}
</style>
