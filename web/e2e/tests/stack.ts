import { createHash, createHmac, randomBytes } from 'node:crypto';
import { expect, request, type APIRequestContext, type Page } from '@playwright/test';

/** Where scripts/e2e.sh raised the stack. */
export const ID = required('E2E_ID_URL');
export const CONSOLE = required('E2E_CONSOLE_URL');
export const SETUP_TOKEN = required('E2E_SETUP_TOKEN');

/** The first administrator, made by setup.spec.ts. */
export const ROOT = { email: 'root@example.com', password: 'root-password-123', firstName: 'Root' };

function required(name: string): string {
	const value = process.env[name];
	if (!value) throw new Error(`${name} is not set: run the tests with make e2e`);
	return value;
}

/**
 * The admin API as the first administrator, the way the panel calls it: on the
 * console's origin, with its Origin, as JSON.
 */
export async function adminAPI(): Promise<APIRequestContext> {
	const api = await request.newContext({
		baseURL: `${CONSOLE}/api/v1/admin/`,
		extraHTTPHeaders: { Origin: CONSOLE }
	});
	const login = await api.post('auth/login', {
		data: { username: ROOT.email, password: ROOT.password }
	});
	expect(login.status(), await login.text()).toBe(200);
	return api;
}

/** Calls the admin API and returns the answer, failing on any other status. */
export async function call<T>(
	api: APIRequestContext,
	method: 'GET' | 'POST' | 'PATCH',
	path: string,
	data?: unknown,
	status = 200
): Promise<T> {
	const answer = await api.fetch(path, { method, data });
	expect(answer.status(), `${method} ${path}: ${await answer.text()}`).toBe(status);
	return (await answer.json()) as T;
}

/** A person made through the admin API, with a proved address. */
export async function makeUser(api: APIRequestContext, email: string, password: string) {
	await call(
		api,
		'POST',
		'users',
		{
			email,
			first_name: 'Grace',
			last_name: 'Hopper',
			password,
			confirm_password: password,
			is_email_verified: true
		},
		201
	);
}

/** Sets the default login flow's steps. */
export async function withSteps(api: APIRequestContext, ...steps: string[]) {
	const { flows } = await call<{ flows: { id: string; is_default: boolean }[] }>(
		api,
		'GET',
		'login-flows'
	);
	const flow = flows.find((f) => f.is_default);
	if (!flow) throw new Error('no default login flow');
	await call(api, 'PATCH', `login-flows/${flow.id}`, { steps });
}

/** A PKCE pair (RFC 7636): the verifier, and its S256 challenge. */
export function pkce() {
	const verifier = randomBytes(32).toString('base64url');
	return { verifier, challenge: createHash('sha256').update(verifier).digest('base64url') };
}

/** The authenticator code for a base32 key at a time (RFC 6238: SHA-1, 30 s, six digits). */
export function totp(key: string, at = Date.now()): string {
	const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';
	let bits = '';
	for (const char of key.replace(/[\s=]/g, '').toUpperCase())
		bits += alphabet.indexOf(char).toString(2).padStart(5, '0');
	const secret = Buffer.from(bits.match(/.{8}/g)!.map((byte) => parseInt(byte, 2)));

	const counter = Buffer.alloc(8);
	counter.writeBigUInt64BE(BigInt(Math.floor(at / 30_000)));
	const mac = createHmac('sha1', secret).update(counter).digest();
	const offset = mac[mac.length - 1] & 0xf;
	return String((mac.readUInt32BE(offset) & 0x7fffffff) % 1_000_000).padStart(6, '0');
}

/**
 * Opens a page and waits until it has hydrated: a field filled in before the
 * app's script runs is a value its state never hears of.
 */
export async function visit(page: Page, url: string) {
	await page.goto(url);
	await page.waitForLoadState('networkidle');
}

/** Signs in on the sign-in pages: the address, the password, Continue. */
export async function signIn(page: Page, email: string, password: string) {
	await page.getByLabel('Email').fill(email);
	await page.getByRole('textbox', { name: 'Password', exact: true }).fill(password);
	await page.getByRole('button', { name: 'Continue' }).click();
}
