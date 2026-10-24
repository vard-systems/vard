<script lang="ts">
	import { PasswordForm, SessionList, TwoStepPanel } from '#lib/components/account/index.js';
	import { PageHeading, PageTitle, Stack } from '#lib/components/ui/index.js';
	import { useTranslator } from '#lib/i18n/index.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();
</script>

<PageTitle parts={[t('security.head'), t('account.nav')]} />

<Stack gap="lg">
	<PageHeading title={t('security.title')} description={t('security.description')} />
	<PasswordForm />
	<TwoStepPanel twoStep={data.twoStep} timezone={data.timezone} />
	<SessionList sessions={data.sessions} timezone={data.timezone} />
</Stack>
