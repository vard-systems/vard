/**
 * The headers every page carries, beside the Content Security Policy SvelteKit writes from
 * vite.config.ts. Every page is drawn for one person (their account, a form with a one-time token
 * in it, data the API sent), so no cache may keep it, and Back after signing out does not show it.
 */
export function setPageHeaders(headers: Headers, url: URL): void {
	// Over https (whoever ends TLS in front), browsers are held to it for two
	// years; over plain http a browser ignores it, so none is sent.
	if (url.protocol === 'https:') headers.set('Strict-Transport-Security', 'max-age=63072000');
	headers.set('X-Frame-Options', 'DENY');
	headers.set('X-Content-Type-Options', 'nosniff');
	headers.set('Referrer-Policy', 'no-referrer');
	// Nothing here uses a camera, a microphone or a payment sheet, so no page
	// may, even an injected one.
	headers.set(
		'Permissions-Policy',
		'accelerometer=(), autoplay=(), camera=(), display-capture=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), screen-wake-lock=(), usb=(), xr-spatial-tracking=()'
	);

	if (headers.get('content-type')?.startsWith('text/html')) {
		headers.set('Cache-Control', 'no-store');
	}
}
