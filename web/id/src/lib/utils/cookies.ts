/** The value of one of this app's own cookies, or undefined. Only cookies
 *  the app sets itself can be read: the session cookie is HttpOnly. */
export function readCookie(name: string): string | undefined {
	for (const pair of document.cookie.split('; ')) {
		const [key, ...value] = pair.split('=');
		if (key === name) return decodeURIComponent(value.join('='));
	}
	return undefined;
}
