import type { IconData } from '#lib/icons.js';

export type Column = {
	key: string;
	label?: string;
	icon?: IconData;
	min?: string;
	align?: 'start' | 'end';
};
