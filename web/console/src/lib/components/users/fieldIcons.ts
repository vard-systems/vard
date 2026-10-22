import type { IconName } from '#lib/components/ui/index.js';
import type { FieldType } from '#lib/api/index.js';

export const fieldIcons: Record<FieldType, IconName> = {
	text: 'ri-text',
	number: 'ri-hashtag',
	bool: 'ri-toggle-line',
	email: 'ri-mail-line',
	date: 'ri-calendar-line'
};
