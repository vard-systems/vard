import type { IconData } from '#lib/icons.js';

export type SelectOption<Value extends string = string> = {
	value: Value;
	label?: string;
	description?: string;
	icon?: IconData;
	disabled?: boolean;
};
