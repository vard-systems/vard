import type { RouteId } from '$app/types';
import type { ComponentType } from 'svelte';
import {
	RiAdminLine,
	RiAppsLine,
	RiBuildingLine,
	RiCodeBoxLine,
	RiComputerLine,
	RiDatabase2Line,
	RiFileList3Line,
	RiGitBranchLine,
	RiGroupLine,
	RiKeyLine,
	RiLinksLine,
	RiMailLine,
	RiPulseLine,
	RiShareLine,
	RiShieldKeyholeLine,
	RiShieldUserLine
} from 'svelte-remixicon';
import type { Admin } from '#lib/api/index.js';
import { can, canAnywhere } from '#lib/permissions.js';

export type Section = Extract<
	Exclude<RouteId, `${string}[${string}`>,
	`/admin/(panel)/dashboard${string}`
>;

export type SidebarItem = {
	route: Section;
	label: string;
	icon: ComponentType;
	allowed?: (admin: Admin) => boolean;
};

export type SidebarGroup = {
	/** A name the cookie can hold, so which groups are folded survives a
	    reload. It is not the label: renaming a group should not unfold it. */
	id: string;
	label: string;
	items: SidebarItem[];
};

const groups: SidebarGroup[] = [
	{
		id: 'overview',
		label: 'Overview',
		items: [
			{
				route: '/admin/(panel)/dashboard',
				label: 'Activity',
				icon: RiPulseLine,
				allowed: (admin) => can(admin, 'activity.read')
			},
			{
				route: '/admin/(panel)/dashboard/logs',
				label: 'Logs',
				icon: RiFileList3Line,
				allowed: (admin) => can(admin, 'activity.read')
			}
		]
	},
	{
		id: 'applications',
		label: 'Applications',
		items: [
			{
				route: '/admin/(panel)/dashboard/applications',
				label: 'Applications',
				icon: RiAppsLine,
				allowed: (admin) => canAnywhere(admin, 'applications.read')
			},
			{
				route: '/admin/(panel)/dashboard/apis',
				label: 'APIs',
				icon: RiCodeBoxLine,
				allowed: (admin) => can(admin, 'apis.read')
			}
		]
	},
	{
		id: 'authentication',
		label: 'Authentication',
		items: [
			{
				route: '/admin/(panel)/dashboard/flows',
				label: 'Login flows',
				icon: RiGitBranchLine,
				allowed: (admin) => can(admin, 'login_flows.read')
			},
			{
				route: '/admin/(panel)/dashboard/social',
				label: 'Social sign-in',
				icon: RiShareLine,
				allowed: (admin) => can(admin, 'social.read')
			},
			{
				route: '/admin/(panel)/dashboard/sso',
				label: 'SSO integrations',
				icon: RiLinksLine,
				allowed: (admin) => can(admin, 'sso.read')
			},
			{
				route: '/admin/(panel)/dashboard/otp',
				label: 'One-time codes',
				icon: RiKeyLine,
				allowed: (admin) => admin.is_super_admin
			}
		]
	},
	{
		id: 'users',
		label: 'User management',
		items: [
			{
				route: '/admin/(panel)/dashboard/users',
				label: 'Users',
				icon: RiGroupLine,
				allowed: (admin) => can(admin, 'users.read')
			},
			{
				route: '/admin/(panel)/dashboard/sessions',
				label: 'Sessions',
				icon: RiComputerLine,
				allowed: (admin) => can(admin, 'users.read')
			},
			{
				route: '/admin/(panel)/dashboard/roles',
				label: 'Roles',
				icon: RiShieldUserLine,
				allowed: (admin) =>
					canAnywhere(admin, 'users.read') || canAnywhere(admin, 'applications.read')
			}
		]
	},
	{
		id: 'administration',
		label: 'Administration',
		items: [
			{
				route: '/admin/(panel)/dashboard/admins',
				label: 'Administrators',
				icon: RiAdminLine,
				allowed: (admin) => admin.is_super_admin
			},
			{
				route: '/admin/(panel)/dashboard/admin-roles',
				label: 'Admin roles',
				icon: RiShieldKeyholeLine,
				allowed: (admin) => admin.is_super_admin
			}
		]
	},
	{
		id: 'settings',
		label: 'Settings',
		items: [
			{
				route: '/admin/(panel)/dashboard/settings',
				label: 'General',
				icon: RiBuildingLine,
				allowed: (admin) => can(admin, 'organization.read') || can(admin, 'languages.read')
			},
			{
				route: '/admin/(panel)/dashboard/mail',
				label: 'Mail',
				icon: RiMailLine,
				allowed: (admin) => admin.is_super_admin
			},
			{
				route: '/admin/(panel)/dashboard/cache',
				label: 'Cache',
				icon: RiDatabase2Line,
				allowed: (admin) => admin.is_super_admin
			}
		]
	}
];

/**
 * Whether `route` is the page being read, compared by route id (resolved paths are relative
 * since SvelteKit 2 and never match the pathname). Activity matches only exactly; others also
 * match pages below them.
 */
export function isCurrentSection(route: Section, id: string | null | undefined): boolean {
	return route === '/admin/(panel)/dashboard'
		? id === route
		: id === route || Boolean(id?.startsWith(`${route}/`));
}

export function visibleGroups(admin: Admin | undefined): SidebarGroup[] {
	return groups
		.map((group) => ({
			...group,
			items: group.items.filter((item) => !item.allowed || (admin && item.allowed(admin)))
		}))
		.filter((group) => group.items.length > 0);
}
