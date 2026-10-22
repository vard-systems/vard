import type { RouteId } from '$app/types';
import type { IconName } from '#lib/components/ui/index.js';
import type { Admin } from '#lib/api/index.js';
import { can, canAnywhere } from '#lib/permissions.js';

export type Section = Extract<Exclude<RouteId, `${string}[${string}`>, `/(panel)${string}`>;

export type SidebarItem = {
	route: Section;
	label: string;
	icon: IconName;
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
				route: '/(panel)',
				label: 'Activity',
				icon: 'ri-pulse-line',
				allowed: (admin) => can(admin, 'activity.read')
			},
			{
				route: '/(panel)/logs',
				label: 'Logs',
				icon: 'ri-file-list-3-line',
				allowed: (admin) => can(admin, 'activity.read')
			}
		]
	},
	{
		id: 'applications',
		label: 'Applications',
		items: [
			{
				route: '/(panel)/applications',
				label: 'Applications',
				icon: 'ri-apps-line',
				allowed: (admin) => canAnywhere(admin, 'applications.read')
			},
			{
				route: '/(panel)/apis',
				label: 'APIs',
				icon: 'ri-code-box-line',
				allowed: (admin) => can(admin, 'apis.read')
			}
		]
	},
	{
		id: 'authentication',
		label: 'Authentication',
		items: [
			{
				route: '/(panel)/flows',
				label: 'Login flows',
				icon: 'ri-git-branch-line',
				allowed: (admin) => can(admin, 'login_flows.read')
			},
			{
				route: '/(panel)/social',
				label: 'Social sign-in',
				icon: 'ri-share-line',
				allowed: (admin) => can(admin, 'social.read')
			},
			{
				route: '/(panel)/sso',
				label: 'SSO integrations',
				icon: 'ri-links-line',
				allowed: (admin) => can(admin, 'sso.read')
			},
			{
				route: '/(panel)/otp',
				label: 'One-time codes',
				icon: 'ri-key-line',
				allowed: (admin) => admin.is_super_admin
			}
		]
	},
	{
		id: 'users',
		label: 'User management',
		items: [
			{
				route: '/(panel)/users',
				label: 'Users',
				icon: 'ri-group-line',
				allowed: (admin) => can(admin, 'users.read')
			},
			{
				route: '/(panel)/sessions',
				label: 'Sessions',
				icon: 'ri-computer-line',
				allowed: (admin) => can(admin, 'users.read')
			},
			{
				route: '/(panel)/roles',
				label: 'Roles',
				icon: 'ri-shield-user-line',
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
				route: '/(panel)/admins',
				label: 'Administrators',
				icon: 'ri-admin-line',
				allowed: (admin) => admin.is_super_admin
			},
			{
				route: '/(panel)/admin-roles',
				label: 'Admin roles',
				icon: 'ri-shield-keyhole-line',
				allowed: (admin) => admin.is_super_admin
			}
		]
	},
	{
		id: 'settings',
		label: 'Settings',
		items: [
			{
				route: '/(panel)/settings',
				label: 'General',
				icon: 'ri-building-line',
				allowed: (admin) => can(admin, 'organization.read') || can(admin, 'languages.read')
			},
			{
				route: '/(panel)/mail',
				label: 'Mail',
				icon: 'ri-mail-line',
				allowed: (admin) => admin.is_super_admin
			},
			{
				route: '/(panel)/cache',
				label: 'Cache',
				icon: 'ri-database-2-line',
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
	return route === '/(panel)' ? id === route : id === route || Boolean(id?.startsWith(`${route}/`));
}

export function visibleGroups(admin: Admin | undefined): SidebarGroup[] {
	return groups
		.map((group) => ({
			...group,
			items: group.items.filter((item) => !item.allowed || (admin && item.allowed(admin)))
		}))
		.filter((group) => group.items.length > 0);
}
