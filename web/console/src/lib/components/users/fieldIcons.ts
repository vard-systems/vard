import {
	RiCalendarLine,
	RiHashtag,
	RiKey2Line,
	RiMailLine,
	RiText,
	RiToggleLine,
	type IconData
} from '#lib/icons.js';
import type { FieldType } from '#lib/api/index.js';

export const fieldIcons: Record<FieldType, IconData> = {
	text: RiText,
	number: RiHashtag,
	bool: RiToggleLine,
	email: RiMailLine,
	date: RiCalendarLine
};

export const idIcon = RiKey2Line;
