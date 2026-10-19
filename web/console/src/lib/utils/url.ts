import { goto } from '$app/navigation';
import { page } from '$app/state';

/**
 * Writes `changes` into the current page's query string and goes there, keeping the page's
 * state. An empty value removes its parameter. A list keeps its filters in its address, so a
 * reload or a shared link shows the same list.
 */
export async function setSearchParams(changes: Record<string, string | undefined>): Promise<void> {
	const url = new URL(page.url.href);

	for (const [key, value] of Object.entries(changes)) {
		if (value) url.searchParams.set(key, value);
		else url.searchParams.delete(key);
	}

	// The page's own address, changed only in its query.
	// eslint-disable-next-line svelte/no-navigation-without-resolve
	await goto(url, { reset: false });
}

/** `run`, called once `wait` milliseconds have passed without another call. */
export function debounce<Args extends unknown[]>(
	run: (...args: Args) => unknown,
	wait = 250
): (...args: Args) => void {
	let timer: ReturnType<typeof setTimeout> | undefined;

	return (...args) => {
		clearTimeout(timer);
		timer = setTimeout(() => run(...args), wait);
	};
}
