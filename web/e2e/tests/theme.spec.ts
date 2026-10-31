import { expect, test, type Page } from '@playwright/test';
import { CONSOLE, ID, visit } from './stack.js';

const html = (page: Page) => page.locator('html');

// A chosen theme is the server's to draw: the page arrives in it, so a
// reload never flashes back to the other one.
test('a chosen theme survives a reload, in each app alone', async ({ page }) => {
	await page.emulateMedia({ colorScheme: 'light' });

	await visit(page, `${ID}/login`);
	await page.getByRole('button', { name: 'Switch theme' }).click();
	await expect(html(page)).toHaveAttribute('data-theme', 'dark');

	await page.reload();
	await expect(html(page)).toHaveAttribute('data-theme', 'dark');
	const rendered = await (await page.request.get(`${ID}/login`)).text();
	expect(rendered).toContain('data-theme="dark"');

	// Every port on localhost shares its cookies; the panel reads only its own.
	await visit(page, `${CONSOLE}/login`);
	await expect(html(page)).not.toHaveAttribute('data-theme', 'dark');

	await page.getByRole('button', { name: 'Switch between the light and dark theme' }).click();
	await expect(html(page)).toHaveAttribute('data-theme', 'dark');
	await page.getByRole('button', { name: 'Switch between the light and dark theme' }).click();
	await expect(html(page)).toHaveAttribute('data-theme', 'light');

	await page.reload();
	await expect(html(page)).toHaveAttribute('data-theme', 'light');

	await visit(page, `${ID}/login`);
	await expect(html(page)).toHaveAttribute('data-theme', 'dark');
});
