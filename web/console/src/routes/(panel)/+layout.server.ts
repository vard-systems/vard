import { adminApi } from '#lib/api/index.js';
import { COOKIES } from '#lib/brand.js';
import { ADMIN_DEPENDENCY } from '#lib/constants.js';
import { fromApi } from '#lib/api/load.js';
import { isSidebarState, parseClosedGroups, type SidebarState } from '#lib/state/sidebar.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies, depends, fetch }) => {
	depends(ADMIN_DEPENDENCY);

	const { admin, organization } = await fromApi(adminApi.me(fetch));

	const saved = cookies.get(COOKIES.sidebar);
	const sidebar: SidebarState = isSidebarState(saved) ? saved : 'wide';
	const closedGroups = parseClosedGroups(cookies.get(COOKIES.branches));

	return { admin, organization, sidebar, closedGroups };
};
