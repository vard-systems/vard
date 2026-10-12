import type { Admin, OrganizationBrand } from '$lib/api';
import { COOKIES } from '$lib/brand';
import { ADMIN_DEPENDENCY } from '$lib/constants';
import { apiGet } from '$lib/server/api';
import { isSidebarState, parseClosedGroups, type SidebarState } from '$lib/state/sidebar';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies, depends, fetch }) => {
	depends(ADMIN_DEPENDENCY);

	const { admin, organization } = await apiGet<{
		admin: Admin;
		organization: OrganizationBrand;
	}>('/admin/me', fetch);

	const saved = cookies.get(COOKIES.sidebar);
	const sidebar: SidebarState = isSidebarState(saved) ? saved : 'wide';
	const closedGroups = parseClosedGroups(cookies.get(COOKIES.branches));

	return { admin, organization, sidebar, closedGroups };
};
