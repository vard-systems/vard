import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

import { currentIcon, groups, isCurrentSection, type Section } from './sections';

describe('isCurrentSection', () => {
	const dashboard = '/(panel)';
	const users = '/(panel)/users';
	const apis = '/(panel)/apis';

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
			id: '/(panel)/users/[id]',
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
			id: '/(panel)/apis-extra',
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
			id: '/users',
			want: false
		}
	];

	for (const { name, route, id, want } of tests) {
		it(`${want ? 'marks' : 'does not mark'} ${name}`, () => {
			expect(isCurrentSection(route, id)).toBe(want);
		});
	}
});

describe('currentIcon', () => {
	const font = readFileSync(
		new URL('../../../../../node_modules/remixicon/fonts/remixicon.css', import.meta.url),
		'utf8'
	);

	it('turns a line icon into its filled twin', () => {
		expect(currentIcon('ri-group-line')).toBe('ri-group-fill');
	});

	it('leaves an icon with no line style as it is', () => {
		expect(currentIcon('ri-hashtag')).toBe('ri-hashtag');
	});

	for (const item of groups.flatMap((group) => group.items)) {
		it(`${item.label}'s filled icon is in the font`, () => {
			expect(font).toContain(`.${currentIcon(item.icon)}:before`);
		});
	}
});
