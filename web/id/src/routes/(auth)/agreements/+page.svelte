<script lang="ts">
	import { signIn } from '#lib/api/index.js';
	import { AuthCard, Decision, LegalLinkList } from '#lib/components/auth/index.js';
	import { Alert, PageTitle } from '#lib/components/ui/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { legalLinks } from '#lib/utils/legal.js';
	import { leaveTo } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const app = $derived(data.application);
	const organization = $derived(data.organization);
	// The organization's own links: the step is about its terms, whatever the
	// application has of its own.
	const links = $derived(legalLinks(null, organization));

	let sending = $state<'yes' | 'no' | null>(null);
	let error = $state('');

	async function answer(yes: boolean) {
		if (!data.openRequest) return;

		error = '';
		sending = yes ? 'yes' : 'no';

		try {
			const { redirect_to } = await signIn.answerAgreements({ request: data.openRequest, yes });
			leaveTo(redirect_to);
		} catch (err) {
			error = messageOf(err, t);
			sending = null;
		}
	}
</script>

<PageTitle parts={[t('agreements.head'), app?.name]} />

{#if !data.openRequest}
	<AuthCard {organization} title={t('agreements.expired_title')}>
		<Alert tone="info">{t('agreements.expired_body')}</Alert>
	</AuthCard>
{:else}
	<AuthCard
		{organization}
		application={app}
		title={t('agreements.title')}
		subtitle={app ? t('agreements.subtitle_app', { app: app.name }) : undefined}
	>
		{#if error}<Alert tone="danger">{error}</Alert>{/if}

		<p>{t('agreements.body', { org: organization?.name ?? '' })}</p>

		<LegalLinkList links={links.map((link) => ({ label: t(link.key), href: link.href }))} />

		<Decision
			yes={t('agreements.accept')}
			no={t('agreements.decline')}
			{sending}
			onanswer={answer}
		/>
	</AuthCard>
{/if}

<style>
	p {
		margin: 0;
	}
</style>
