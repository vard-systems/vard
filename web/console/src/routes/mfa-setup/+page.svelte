<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { adminApi } from '#lib/api/index.js';
	import { BRAND } from '#lib/brand.js';
	import TotpSetup from '#lib/components/mfa/TotpSetup.svelte';
	import { Card, ThemeToggle } from '#lib/components/ui/index.js';

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
	<title>Set up two-factor sign-in · {BRAND.name}</title>
</svelte:head>

<main>
	<div class="corner"><ThemeToggle /></div>

	<div class="panel">
		<Card padded>
			<div class="content">
				<header>
					<span class="mark"
						><i class="ri-shield-check-line" style:font-size="20px" aria-hidden="true"></i></span
					>
					<h1>Set up two-factor sign-in</h1>
					<p class="muted">
						{data.required
							? 'Every administrator signs in with a code from an authenticator app as well as a password. Set yours up to continue.'
							: 'Sign in with a code from an authenticator app as well as your password.'}
					</p>
				</header>

				<TotpSetup
					doneLabel={data.enrolling ? 'Continue to the panel' : 'Done'}
					onDone={done}
					onCancel={cancel}
				/>
			</div>
		</Card>
	</div>
</main>

<style>
	main {
		display: grid;
		place-items: center;
		min-height: 100dvh;
		padding: var(--space-5);
	}

	.corner {
		position: fixed;
		top: var(--space-3);
		right: var(--space-3);
		color: var(--color-text-hint);
	}

	.panel {
		width: 100%;
		max-width: 28rem;
	}

	.content {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
		padding: var(--space-2);
	}

	header {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.mark {
		display: grid;
		place-items: center;
		width: 40px;
		height: 40px;
		margin-bottom: var(--space-2);
		border-radius: var(--radius-md);
		background: var(--color-accent);
		color: var(--color-accent-text);
	}

	h1 {
		font-size: var(--text-lg);
	}

	.muted {
		color: var(--color-text-hint);
		line-height: 1.5;
	}
</style>
