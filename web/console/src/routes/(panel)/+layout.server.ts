import { adminApi } from '#lib/api/index.js';
import { COOKIES } from '#lib/brand.js';
import { ADMIN_DEPENDENCY } from '#lib/constants.js';
import { fromApi } from '#lib/api/load.js';
import { readChoice } from '#lib/utils/cookies.js';
import { isSidebarState, parseClosedGroups, type SidebarState } from '#lib/state/sidebar.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies, depends, fetch, url }) => {
	depends(ADMIN_DEPENDENCY);

	const { admin, organization } = await fromApi(adminApi.me(fetch));

	const saved = readChoice(cookies, COOKIES.sidebar, url);
	const sidebar: SidebarState = isSidebarState(saved) ? saved : 'wide';
	const closedGroups = parseClosedGroups(readChoice(cookies, COOKIES.branches, url));

	return { admin, organization, sidebar, closedGroups };
};
