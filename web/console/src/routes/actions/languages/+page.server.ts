import { languagesApi, type LocaleApp } from '#lib/api/index.js';
import { act, action } from '#lib/server/actions.js';
import type { Actions } from './$types';

type LanguageSettings = Parameters<typeof languagesApi.update>[1];

export const actions = {
	create: action(languagesApi.create),
	// One save is the language's settings, when they changed, then the text of
	// each app whose text did, as the dialog had made them one by one.
	save: (event) =>
		act(
			event,
			async ([code, settings, texts]: [
				string,
				LanguageSettings | null,
				[LocaleApp, Record<string, string>][]
			]) => {
				if (settings) await languagesApi.update(code, settings);
				for (const [app, messages] of texts) {
					await languagesApi.saveTranslation(code, app, messages);
				}
				return { code };
			}
		)
} satisfies Actions;
