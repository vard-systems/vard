import type { IconName } from './icon';

export type SelectOption<Value extends string = string> = {
	value: Value;
	label?: string;
	description?: string;
	icon?: IconName;
	disabled?: boolean;
};
