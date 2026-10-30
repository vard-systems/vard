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
		(r) => r.url().endsWith('/api/v1/admin/users') && r.request().method() === 'POST'
	);
	await dialog.getByRole('button', { name: 'Create user' }).click();
	expect((await created).status()).toBe(201);

	await visit(page, `${CONSOLE}/users`);
	await expect(page.getByText('ada@example.com', { exact: true })).toBeVisible();
});
