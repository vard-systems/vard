import { queryOptions } from '@tanstack/svelte-query';

import { mailApi, type MailContent, type MailResponse } from '#lib/api/index.js';
import { keys } from './keys';

export function mailOptions(initial: MailResponse) {
	return queryOptions({
		queryKey: keys.mail.settings,
		queryFn: () => mailApi.get(),
		initialData: initial
	});
}

export function mailContentOptions(initial: MailContent) {
	return queryOptions({
		queryKey: keys.mail.content,
		queryFn: () => mailApi.content(),
		initialData: initial
	});
}
