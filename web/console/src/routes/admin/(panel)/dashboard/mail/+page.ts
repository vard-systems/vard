import type { MailContent, MailResponse } from '#lib/api/index.js';
import { apiGet, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	requirePermission((await parent()).admin, 'super_admin');

	const [mail, content] = await Promise.all([
		apiGet<MailResponse>('/admin/mail', fetch),
		apiGet<MailContent>('/admin/mail/content', fetch)
	]);

	return { mail, content };
};
