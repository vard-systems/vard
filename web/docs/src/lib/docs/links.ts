import { resolve } from '$app/paths';
import type { PathnameWithSearchOrHash } from '$app/types';

/**
 * Prefixes a docs path with the base path; the one place a stored address becomes a link.
 * Stored addresses start with a slash, which resolve() reads as a route id, so it goes.
 */
export function href(path: string): string {
	return resolve(path.replace(/^\/+/, '') as PathnameWithSearchOrHash);
}
