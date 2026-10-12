import { flatten, type Messages } from './flatten';

export type { Messages } from './flatten';

export const BASE = 'en';

const groupModules = import.meta.glob('../../../../../i18n/id/en/*.json', {
	eager: true,
	import: 'default'
});

export const base: Messages = Object.assign(
	{},
	...Object.values(groupModules).map((file) => flatten(file) ?? {})
);
