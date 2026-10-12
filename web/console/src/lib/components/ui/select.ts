import type { ComponentType } from 'svelte';

export type SelectOption<Value extends string = string> = {
	value: Value;
	label?: string;
	description?: string;
	icon?: ComponentType;
	disabled?: boolean;
};
