import type { MailContent, MailResponse } from '$lib/api';
import { apiGet, requirePermission } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent }) => {
	requirePermission((await parent()).admin, 'super_admin');

	const [mail, content] = await Promise.all([
		apiGet<MailResponse>('/admin/mail', fetch),
		apiGet<MailContent>('/admin/mail/content', fetch)
	]);

	return { mail, content };
};
