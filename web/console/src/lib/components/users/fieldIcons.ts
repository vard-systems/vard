import type { ComponentType } from 'svelte';
import {
	RiCalendarLine,
	RiHashtag,
	RiKey2Line,
	RiMailLine,
	RiText,
	RiToggleLine
} from 'svelte-remixicon';
import type { FieldType } from '#lib/api/index.js';

export const fieldIcons: Record<FieldType, ComponentType> = {
	text: RiText,
	number: RiHashtag,
	bool: RiToggleLine,
	email: RiMailLine,
	date: RiCalendarLine
};

export const idIcon = RiKey2Line;
