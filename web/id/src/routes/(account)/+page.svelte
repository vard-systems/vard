<script lang="ts">
	import { AccountDetails, ProfileForm, ProfileIntro } from '#lib/components/account/index.js';
	import { PageTitle, Stack } from '#lib/components/ui/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();
</script>

<PageTitle parts={[t('profile.head'), t('account.nav')]} />

<Stack gap="lg">
	<ProfileIntro user={data.user} />
	<ProfileForm user={data.user} mayChangeEmail={data.login?.allow_email_change ?? false} />
	<AccountDetails user={data.user} timezone={data.timezone} />
</Stack>
