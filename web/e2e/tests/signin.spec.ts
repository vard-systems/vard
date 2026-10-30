import { expect, test, request } from '@playwright/test';
import { adminAPI, call, ID, makeUser, pkce, signIn, totp, withSteps, visit } from './stack.js';

const CALLBACK = 'https://shop.example.test/callback';
const GRACE = { email: 'grace@example.com', password: 'grace-password-123' };

let client: { id: string; secret: string };

test.beforeAll(async () => {
	const api = await adminAPI();
	await withSteps(api, 'identifier', 'password', 'social', 'totp');
	await makeUser(api, GRACE.email, GRACE.password);

	const made = await call<{ application: { client_id: string }; client_secret: string }>(
		api,
		'POST',
		'applications',
		{
			name: 'Shop',
			type: 'web',
			grant_types: ['authorization_code', 'refresh_token'],
			redirect_uris: [CALLBACK],
			scopes: ['openid', 'profile', 'email']
		},
		201
	);
	client = { id: made.application.client_id, secret: made.client_secret };
	await api.dispose();
});

/** Starts a sign-in to the shop, and answers the redirect back to it here. */
async function authorize(page: import('@playwright/test').Page) {
	const { verifier, challenge } = pkce();
	await page.route(`${CALLBACK}*`, (route) => route.fulfill({ body: 'back at the shop' }));
	const query = new URLSearchParams({
		response_type: 'code',
		client_id: client.id,
		redirect_uri: CALLBACK,
		scope: 'openid profile email',
		state: 'e2e-state',
		nonce: 'e2e-nonce',
		code_challenge: challenge,
		code_challenge_method: 'S256'
	});
	await visit(page, `${ID}/oauth2/authorize?${query}`);
	return verifier;
}

/** Waits for the shop's callback and redeems its code for tokens. */
async function redeem(page: import('@playwright/test').Page, verifier: string) {
	await page.waitForURL(`${CALLBACK}*`);
	const back = new URL(page.url());
	expect(back.searchParams.get('state')).toBe('e2e-state');

	const token = await request.newContext();
	const answer = await token.post(`${ID}/oauth2/token`, {
		headers: {
			Authorization: `Basic ${Buffer.from(`${client.id}:${client.secret}`).toString('base64')}`
		},
		form: {
			grant_type: 'authorization_code',
			code: back.searchParams.get('code')!,
			redirect_uri: CALLBACK,
			code_verifier: verifier
		}
	});
	expect(answer.status(), await answer.text()).toBe(200);
	const { id_token } = await answer.json();
	return JSON.parse(Buffer.from(id_token.split('.')[1], 'base64url').toString()) as Record<
		string,
		unknown
	>;
}

// A person signs in to an application with a password and is sent back with a
// code the application redeems for an ID token about them.
test('a person signs in to an application', async ({ page }) => {
	const verifier = await authorize(page);
	await expect(page.getByRole('heading', { name: 'Sign in' })).toBeVisible();
	await signIn(page, GRACE.email, GRACE.password);

	const claims = await redeem(page, verifier);
	expect(claims.email).toBe(GRACE.email);
	expect(claims.nonce).toBe('e2e-nonce');
	expect(claims.amr).toEqual(['pwd']);
});

// The account pages show the person who is signed in.
test('the account pages show who is signed in', async ({ page }) => {
	await visit(page, `${ID}/login`);
	await signIn(page, GRACE.email, GRACE.password);
	await expect(page.getByText(GRACE.email).first()).toBeVisible();
});

// Someone who turns on an authenticator on the security page is asked for it
// at their next sign-in, and the application learns it was multi-factor.
test('an authenticator is asked for once it is turned on', async ({ page, browser }) => {
	await visit(page, `${ID}/login`);
	await signIn(page, GRACE.email, GRACE.password);
	await expect(page).not.toHaveURL(/\/login/);

	await visit(page, `${ID}/security`);
	await page.getByRole('button', { name: 'Set up' }).click();
	await page.getByLabel('Your password, to confirm').fill(GRACE.password);
	await page.getByRole('button', { name: 'Continue' }).click();

	const keyField = page.getByLabel('Key');
	await expect(keyField).not.toHaveValue('');
	const key = await keyField.inputValue();
	await page.getByLabel('Code from the app').fill(totp(key));
	await page.getByRole('button', { name: 'Turn on' }).click();
	await page.getByRole('button', { name: 'I’ve saved them' }).click();
	await expect(page.getByText('On', { exact: true })).toBeVisible();

	const fresh = await (await browser.newContext()).newPage();
	const verifier = await authorize(fresh);
	await signIn(fresh, GRACE.email, GRACE.password);
	await expect(fresh.getByRole('heading', { name: 'Confirm it’s you' })).toBeVisible();
	// The code just used is spent; the next step's is good, within the skew.
	await fresh.getByLabel('Code from your authenticator app').fill(totp(key, Date.now() + 30_000));
	await fresh.getByRole('button', { name: 'Continue' }).click();

	const claims = await redeem(fresh, verifier);
	expect(claims.amr).toEqual(['pwd', 'otp', 'mfa']);
});
