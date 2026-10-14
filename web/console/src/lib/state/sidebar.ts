import { COOKIES } from '#lib/brand.js';

export type SidebarState = 'wide' | 'mini';

const ONE_YEAR = 60 * 60 * 24 * 365;

export function isSidebarState(value: unknown): value is SidebarState {
	return value === 'wide' || value === 'mini';
}

export function rememberSidebar(state: SidebarState) {
	document.cookie = `${COOKIES.sidebar}=${state}; path=/; max-age=${ONE_YEAR}; samesite=lax`;
}

export function parseClosedGroups(raw: string | undefined): string[] {
	if (!raw) return [];

	return raw
		.split(',')
		.map((id) => id.trim())
		.filter((id) => /^[a-z-]+$/.test(id));
}

export function rememberClosedGroups(closed: string[]) {
	const value = closed.join(',');

	document.cookie = `${COOKIES.branches}=${encodeURIComponent(value)}; path=/; max-age=${ONE_YEAR}; samesite=lax`;
}
