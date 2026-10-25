<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApi } from '#lib/api/index.js';
	import { pageTitle } from '#lib/brand.js';
	import GateShell from '#lib/components/layout/GateShell.svelte';
	import TotpSetup from '#lib/components/mfa/TotpSetup.svelte';

	let { data } = $props();

	async function done() {
		await goto(resolve('/(panel)'), { replace: true, refreshAll: true });
	}

	async function cancel() {
		if (data.enrolling) {
			// Half signed in: leaving without a factor means signing out.
			await adminApi.logout().catch(() => {});
			await goto(resolve('/login'), { replace: true, refreshAll: true });
		} else {
			await goto(resolve('/(panel)'));
		}
	}
</script>

<svelte:head>
	<title>{pageTitle('Set up two-factor sign-in')}</title>
</svelte:head>

<GateShell
	title="Set up two-factor sign-in"
	icon="ri-shield-check-line"
	width="md"
	description={data.required
		? 'Every administrator signs in with a code from an authenticator app as well as a password. Set yours up to continue.'
		: 'Sign in with a code from an authenticator app as well as your password.'}
>
	<TotpSetup
		doneLabel={data.enrolling ? 'Continue to the panel' : 'Done'}
		onDone={done}
		onCancel={cancel}
	/>
</GateShell>
