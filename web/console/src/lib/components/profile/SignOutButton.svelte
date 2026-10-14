<script lang="ts">
	import { goto, refreshAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { useQueryClient } from '@tanstack/svelte-query';
	import { RiLogoutBoxRLine } from 'svelte-remixicon';
	import { adminApi } from '#lib/api/index.js';
	import { Button, type Variant } from '#lib/components/ui/index.js';

	type Props = { variant?: Variant };

	let { variant = 'subtle' }: Props = $props();

	const queryClient = useQueryClient();

	let signingOut = $state(false);

	export async function signOut() {
		signingOut = true;

		try {
			await adminApi.logout();
		} finally {
			// However the server answered, this browser is done with the
			// session: drop everything that was loaded with it — the pages
			// and the cache behind them — and go to the sign-in page. What
			// one administrator saw is not for whoever signs in next.
			queryClient.clear();
			await refreshAll();
			await goto(resolve('admin/login'), { replace: true });
		}
	}
</script>

<Button {variant} icon={RiLogoutBoxRLine} loading={signingOut} onclick={signOut}
	>{signingOut ? 'Signing out…' : 'Sign out'}</Button
>
