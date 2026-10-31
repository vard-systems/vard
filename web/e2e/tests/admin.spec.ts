import { expect, test } from '@playwright/test';
import { CONSOLE, ROOT, visit } from './stack.js';

// An administrator signs in to the panel and makes a user, who is then listed.
test('an administrator creates a user', async ({ page }) => {
	await visit(page, `${CONSOLE}/login`);
	await page.getByLabel('Email').fill(ROOT.email);
	await page.getByRole('textbox', { name: 'Password', exact: true }).fill(ROOT.password);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Activity' })).toBeVisible();

	await visit(page, `${CONSOLE}/users`);
	await page.getByRole('button', { name: 'New user' }).click();

	const dialog = page.getByRole('dialog');
	await dialog.getByRole('textbox', { name: 'Email' }).fill('ada@example.com');
	await dialog.getByLabel('First name').fill('Ada');
	await dialog.getByRole('textbox', { name: 'Password', exact: true }).fill('ada-password-123');
	await dialog.getByLabel('Confirm password').fill('ada-password-123');
	const created = page.waitForResponse(
		(r) => r.url().includes('/actions/users?/create') && r.request().method() === 'POST'
	);
	await dialog.getByRole('button', { name: 'Create user' }).click();
	expect((await created).status()).toBe(200);
	await expect(page.getByText('User created', { exact: true })).toBeVisible();

	await visit(page, `${CONSOLE}/users`);
	await expect(page.getByText('ada@example.com', { exact: true })).toBeVisible();
});

// A setting is saved through the panel's form action, which asks the admin
// API with the administrator's own session.
test('an administrator renames the organization', async ({ page }) => {
	await visit(page, `${CONSOLE}/login`);
	await page.getByLabel('Email').fill(ROOT.email);
	await page.getByRole('textbox', { name: 'Password', exact: true }).fill(ROOT.password);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Activity' })).toBeVisible();

	await visit(page, `${CONSOLE}/settings`);
	await page.getByRole('textbox', { name: 'Name', exact: true }).fill('Acme');
	await page.getByRole('button', { name: 'Save changes' }).click();
	await expect(page.getByText('Settings saved')).toBeVisible();

	await visit(page, `${CONSOLE}/settings`);
	await expect(page.getByRole('textbox', { name: 'Name', exact: true })).toHaveValue('Acme');
});

// A form action vouches to the admin API for the console's origin, so one
// posted from anywhere else — with the administrator's cookies, as a page on
// another site could — is refused before it runs.
test('a form posted from another site is refused', async ({ page }) => {
	await visit(page, `${CONSOLE}/login`);
	await page.getByLabel('Email').fill(ROOT.email);
	await page.getByRole('textbox', { name: 'Password', exact: true }).fill(ROOT.password);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Activity' })).toBeVisible();

	const forged = await page.request.post(`${CONSOLE}/actions/organization?/update`, {
		headers: { Origin: 'https://evil.example' },
		form: { input: JSON.stringify({ name: 'Evil' }) }
	});
	expect(forged.status()).toBe(403);

	await visit(page, `${CONSOLE}/settings`);
	await expect(page.getByRole('textbox', { name: 'Name', exact: true })).not.toHaveValue('Evil');
});

// Pages arrive rendered: the HTML the server sends already holds the data,
// before any script runs, as SvelteKit renders it with the page's load.
test('the panel arrives rendered, with its data', async ({ page }) => {
	await visit(page, `${CONSOLE}/login`);
	await page.getByLabel('Email').fill(ROOT.email);
	await page.getByRole('textbox', { name: 'Password', exact: true }).fill(ROOT.password);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Activity' })).toBeVisible();

	const html = await (await page.request.get(`${CONSOLE}/users`)).text();
	expect(html).toContain('ada@example.com');
	expect(html).toContain('<title>');
});
