import type { ComponentType } from 'svelte';

export type Column = {
	key: string;
	label?: string;
	icon?: ComponentType;
	min?: string;
	align?: 'start' | 'end';
};
