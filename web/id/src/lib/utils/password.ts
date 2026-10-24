import { LIMITS } from '#lib/constants.js';

/** Counted in characters, as the server counts them: "é" or an emoji is one,
 *  where `String.length` would count some as two. */
export function longEnough(password: string): boolean {
	return [...password].length >= LIMITS.passwordMin;
}

/** A new password the form may send: long enough, and typed the same twice. */
export function newPasswordReady(password: string, confirm: string): boolean {
	return longEnough(password) && confirm === password;
}
