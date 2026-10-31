import { expect, test } from '@playwright/test';
import { CONSOLE, ROOT, visit } from './stack.js';

// A fresh installation has no administrator; the first is made on the setup
// page, and is signed straight into the panel.
test('the first administrator is made on the setup page', async ({ page }) => {
	await visit(page, `${CONSOLE}/new-super-admin`);

	await page.getByLabel('First name').fill(ROOT.firstName);
	await page.getByLabel('Email').fill(ROOT.email);
	await page.getByRole('textbox', { name: 'Password', exact: true }).fill(ROOT.password);
	await page.getByRole('button', { name: 'Create the super admin' }).click();

	await expect(page).not.toHaveURL(/new-super-admin|login/);
	await expect(page.getByRole('heading', { level: 1, name: 'Activity' })).toBeVisible();
});
