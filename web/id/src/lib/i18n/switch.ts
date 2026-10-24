import { invalidate } from '$app/navigation';
import { tick } from 'svelte';

import { COOKIES, LANGUAGE_DEPENDENCY } from '#lib/brand.js';
import { rememberChoice } from '#lib/utils/cookies.js';

export async function switchLanguage(code: string) {
	rememberChoice(COOKIES.language, code);

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
