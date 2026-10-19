import type { Admin, OrganizationBrand } from '#lib/api/index.js';
import { COOKIES } from '#lib/brand.js';
import { ADMIN_DEPENDENCY } from '#lib/constants.js';
import { apiGet } from '#lib/api/load.js';
import { isSidebarState, parseClosedGroups, type SidebarState } from '#lib/state/sidebar.js';
import { readCookie } from '#lib/utils/cookies.js';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ depends, fetch }) => {
	depends(ADMIN_DEPENDENCY);

	const { admin, organization } = await apiGet<{
		admin: Admin;
		organization: OrganizationBrand;
	}>('/admin/me', fetch);

	const saved = readCookie(COOKIES.sidebar);
	const sidebar: SidebarState = isSidebarState(saved) ? saved : 'wide';
	const closedGroups = parseClosedGroups(readCookie(COOKIES.branches));

	return { admin, organization, sidebar, closedGroups };
};
