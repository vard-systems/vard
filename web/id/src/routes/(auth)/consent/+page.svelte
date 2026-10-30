<script lang="ts">
	import { signIn } from '#lib/api/index.js';
	import { AuthCard, Decision, ScopeList } from '#lib/components/auth/index.js';
	import { Alert, Note, PageTitle } from '#lib/components/ui/index.js';
	import { messageOf, useTranslator } from '#lib/i18n/index.js';
	import { leaveTo } from '#lib/utils/links.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();

	const t = useTranslator();

	const consent = $derived(data.openRequest ? data.consent : null);
	const app = $derived(consent?.application ?? data.application);

	let sending = $state<'yes' | 'no' | null>(null);
	let error = $state('');

	async function answer(yes: boolean) {
		if (!data.openRequest) return;

		error = '';
		sending = yes ? 'yes' : 'no';

		try {
			const { redirect_to } = await signIn.answerConsent({ request: data.openRequest, yes });
			leaveTo(redirect_to);
		} catch (err) {
			error = messageOf(err, t);
			sending = null;
		}
	}
</script>

<PageTitle parts={[t('consent.head'), app?.name]} />

{#if !consent}
	<AuthCard organization={data.organization} title={t('consent.expired_title')}>
		<Alert tone="info">{t('consent.expired_body')}</Alert>
	</AuthCard>
{:else}
	<AuthCard
		organization={data.organization}
		application={app}
		title={t('consent.title', { app: consent.application.name })}
		subtitle={data.user ? t('consent.signed_in_as', { email: data.user.email }) : undefined}
	>
		{#if error}<Alert tone="danger">{error}</Alert>{/if}

		<p class="asks">{t('consent.asks')}</p>

		<ScopeList scopes={consent.scopes} />

		<Decision yes={t('consent.allow')} no={t('consent.deny')} {sending} onanswer={answer} />

		<Note align="center">{t('consent.later')}</Note>
	</AuthCard>
{/if}

<style>
	.asks {
		margin: 0;
		font-weight: 600;
	}
</style>
