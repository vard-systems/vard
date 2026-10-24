import { MIN_ADMIN_PASSWORD } from '#lib/constants.js';

/** Counted in characters, as the server counts them: "é" or an emoji is one,
 *  where `String.length` would count some as two. */
export function longEnoughForAdmin(password: string): boolean {
	return [...password].length >= MIN_ADMIN_PASSWORD;
}
