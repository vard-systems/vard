import { queryOptions } from '@tanstack/svelte-query';

import { languagesApi, type LanguageList, type LocaleApp } from '#lib/api/index.js';
import { keys } from './keys';

export function languagesOptions(initial: LanguageList) {
	return queryOptions({
		queryKey: keys.languages.list,
		queryFn: () => languagesApi.list(),
		initialData: initial
	});
}

export function translationOptions(code: string, app: LocaleApp) {
	return queryOptions({
		queryKey: keys.languages.translation(code, app),
		queryFn: () => languagesApi.translation(code, app),
		staleTime: Infinity
	});
}
