import { invalidate } from '$app/navigation';
import { tick } from 'svelte';

import { COOKIES, LANGUAGE_DEPENDENCY } from '$lib/brand';

const ONE_YEAR = 60 * 60 * 24 * 365;

export async function switchLanguage(code: string) {
	document.cookie = `${COOKIES.language}=${encodeURIComponent(code)}; path=/; max-age=${ONE_YEAR}; samesite=lax`;

	const redraw = async () => {
		await invalidate(LANGUAGE_DEPENDENCY);
		await tick();
	};

	const still = matchMedia('(prefers-reduced-motion: reduce)').matches;
	if (!document.startViewTransition || still || document.hidden) {
		await redraw();
		return;
	}

	const transition = document.startViewTransition(redraw);
	transition.ready.catch(() => {});
	await transition.updateCallbackDone;
}
