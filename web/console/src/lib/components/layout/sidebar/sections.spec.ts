import { describe, expect, it } from 'vitest';

import { isCurrentSection, type Section } from './sections';

describe('isCurrentSection', () => {
	const dashboard = '/admin/(panel)/dashboard';
	const users = '/admin/(panel)/dashboard/users';
	const apis = '/admin/(panel)/dashboard/apis';

	const tests: { name: string; route: Section; id: string | null; want: boolean }[] = [
		{
			name: 'the page itself',
			route: users,
			id: users,
			want: true
		},
		{
			name: 'a page under it, by its own route',
			route: users,
			id: '/admin/(panel)/dashboard/users/[id]',
			want: true
		},
		{
			name: 'another page',
			route: users,
			id: apis,
			want: false
		},
		{
			name: 'a page whose id starts with the same letters',
			route: apis,
			id: '/admin/(panel)/dashboard/apis-extra',
			want: false
		},
		{
			name: 'the dashboard, which is the page of its own row',
			route: dashboard,
			id: dashboard,
			want: true
		},
		{
			name: 'a page under the dashboard, which is not the dashboard',
			route: dashboard,
			id: users,
			want: false
		},
		{
			name: 'no page at all, before the first navigation',
			route: users,
			id: null,
			want: false
		},
		{
			name: 'a path rather than a route id, which is not what this compares',
			route: users,
			id: '/admin/dashboard/users',
			want: false
		}
	];

	for (const { name, route, id, want } of tests) {
		it(`${want ? 'marks' : 'does not mark'} ${name}`, () => {
			expect(isCurrentSection(route, id)).toBe(want);
		});
	}
});
