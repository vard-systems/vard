import { expect, request, test, type Page } from '@playwright/test';
import { adminAPI, call, ID, makeUser, pkce, signIn, totp, visit, withSteps } from './stack.js';

const CALLBACK = 'https://vault.example.test/callback';
const MARGARET = { email: 'margaret@example.com', password: 'margaret-password-123' };
const KATHERINE = { email: 'katherine@example.com', password: 'katherine-password-123' };

let client: { id: string; secret: string };

// An application whose login flow requires a second factor.
test.beforeAll(async () => {
	const api = await adminAPI();
	await withSteps(api, 'identifier', 'password', 'social', 'totp');
	await makeUser(api, MARGARET.email, MARGARET.password);
	await makeUser(api, KATHERINE.email, KATHERINE.password);

	const { flow } = await call<{ flow: { id: string } }>(
		api,
		'POST',
		'login-flows',
		{
			name: 'Vault sign-in',
			slug: 'vault',
			is_enabled: true,
			steps: ['identifier', 'password', 'totp'],
			require_second_factor: true
		},
		201
	);
	const made = await call<{ application: { client_id: string }; client_secret: string }>(
		api,
		'POST',
		'applications',
		{
			name: 'Vault',
			type: 'web',
			grant_types: ['authorization_code'],
			redirect_uris: [CALLBACK],
			scopes: ['openid', 'profile', 'email'],
			login_flow_id: flow.id
		},
		201
	);
	client = { id: made.application.client_id, secret: made.client_secret };
	await api.dispose();
});

async function authorize(page: Page) {
	const { verifier, challenge } = pkce();
	await page.route(`${CALLBACK}*`, (route) => route.fulfill({ body: 'back at the vault' }));
	const query = new URLSearchParams({
		response_type: 'code',
		client_id: client.id,
		redirect_uri: CALLBACK,
		scope: 'openid profile email',
		state: 'vault-state',
		nonce: 'vault-nonce',
		code_challenge: challenge,
		code_challenge_method: 'S256'
	});
	await visit(page, `${ID}/oauth2/authorize?${query}`);
	return verifier;
}

async function redeem(page: Page, verifier: string) {
	await page.waitForURL(`${CALLBACK}*`);
	const back = new URL(page.url());
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

// Someone signed in with a password alone, who has an authenticator, is asked
// for it before an application that requires one, without signing in again.
test('a signed-in person steps up for an application that requires a second factor', async ({
	page
}) => {
	await visit(page, `${ID}/login`);
	await signIn(page, MARGARET.email, MARGARET.password);
	await expect(page).not.toHaveURL(/\/login/);

	await visit(page, `${ID}/security`);
	await page.getByRole('button', { name: 'Set up' }).click();
	await page.getByLabel('Your password, to confirm').fill(MARGARET.password);
	await page.getByRole('button', { name: 'Continue' }).click();
	const keyField = page.getByLabel('Key');
	await expect(keyField).not.toHaveValue('');
	const key = await keyField.inputValue();
	await page.getByLabel('Code from the app').fill(totp(key));
	await page.getByRole('button', { name: 'Turn on' }).click();
	await page.getByRole('button', { name: 'I’ve saved them' }).click();
	await expect(page.getByText('On', { exact: true })).toBeVisible();

	const verifier = await authorize(page);
	await expect(page.getByRole('heading', { name: 'This application asks for more' })).toBeVisible();
	await expect(page.getByText('Vault asks for two-step verification.')).toBeVisible();
	await expect(page.getByLabel('Email')).toHaveCount(0);
	// Somebody signed in has no sign-in to go back to.
	await expect(page.getByRole('link', { name: 'Back to sign in' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Not now' })).toBeVisible();
	await page.getByLabel('Code from your authenticator app').fill(totp(key, Date.now() + 30_000));
	await page.getByRole('button', { name: 'Continue' }).click();

	const claims = await redeem(page, verifier);
	expect(claims.email).toBe(MARGARET.email);
	expect(claims.amr).toEqual(['pwd', 'otp', 'mfa']);
});

// Somebody without a second factor sets one up while signing in to an
// application that requires one, keeps the recovery codes, and arrives.
test('a person without a second factor sets one up while signing in', async ({ page }) => {
	const verifier = await authorize(page);
	await signIn(page, KATHERINE.email, KATHERINE.password);

	await expect(page.getByRole('heading', { name: 'Set up two-step verification' })).toBeVisible();
	await expect(page.getByText('Vault asks for two-step verification.')).toBeVisible();
	await page.getByRole('button', { name: 'Use an authenticator app' }).click();

	const keyField = page.getByLabel('Key');
	await expect(keyField).not.toHaveValue('');
	const key = await keyField.inputValue();
	await page.getByLabel('Code from the app').fill(totp(key));
	await page.getByRole('button', { name: 'Turn on' }).click();

	await expect(page.getByText('Two-step verification is on.')).toBeVisible();
	await expect(page.locator('ol code')).toHaveCount(10);
	await page.getByRole('button', { name: 'I’ve saved them' }).click();

	const claims = await redeem(page, verifier);
	expect(claims.email).toBe(KATHERINE.email);
	expect(claims.amr).toEqual(['pwd', 'otp', 'mfa']);
});
