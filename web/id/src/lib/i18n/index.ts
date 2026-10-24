import { get } from 'svelte/store';
import { createContext } from 'svelte';
import { addMessages, format, getMessageFormatter, init, json, locale } from 'svelte-i18n';
import { ApiError } from '#lib/api/client.js';
import { BASE, base, type Messages } from './messages';

export { BASE, type Messages } from './messages';

export type Translate = ((key: string, params?: Record<string, string | number>) => string) & {
	has: (key: string) => boolean;
	readonly language: string;
};

let initialized = false;
let activeLocale: string | undefined;

function initialize() {
	if (initialized) return;

	void init({
		fallbackLocale: BASE,
		initialLocale: BASE,
		handleMissingMessage: () => undefined
	});
	addMessages(BASE, base);
	initialized = true;
}

/**
 * Builds a translator over `messages()`. svelte-i18n is a singleton, so on the server each
 * request keeps its own messages and uses the low-level formatter; messages are read on every
 * lookup, so pages redraw when the language changes.
 */
export function translator(
	messages: () => Messages,
	language: () => string = () => BASE
): Translate {
	let registeredLanguage: string | undefined;
	let registeredMessages: Messages | undefined;

	const sync = () => {
		const code = language();
		const current = messages();

		// The browser is the one place where the package's global dictionary
		// and locale store are useful: there is one page and one reader there.
		if (typeof window !== 'undefined') {
			initialize();
			if (registeredLanguage !== code || registeredMessages !== current) {
				addMessages(code, current);
				registeredLanguage = code;
				registeredMessages = current;
			}
			if (activeLocale !== code) {
				void locale.set(code);
				activeLocale = code;
			}
		}

		return { code, current };
	};

	const translate = (key: string, params?: Record<string, string | number>) => {
		const { code, current } = sync();
		const fallback = base[key] ?? key;

		if (typeof window === 'undefined') {
			return formatMessage(current[key] || fallback, code, params);
		}

		return get(format)(key, {
			locale: code,
			values: params,
			default: fallback
		});
	};

	const lookup = (key: string) => {
		const { code, current } = sync();
		if (typeof window === 'undefined') return current[key] || base[key];

		return get(json)(key, code);
	};

	return Object.defineProperties(translate, {
		has: { value: (key: string) => Boolean(lookup(key)) },
		language: { get: language }
	}) as Translate;
}

function formatMessage(
	text: string,
	language: string,
	params?: Record<string, string | number>
): string {
	if (!params) return text;

	try {
		return String(getMessageFormatter(text, language).format(params));
	} catch {
		return text;
	}
}

export function messageOf(err: unknown, t: Translate): string {
	if (!(err instanceof ApiError)) return t('error.unknown');

	const key = `error.${err.code}`;
	if (!t.has(key)) return err.message || t('error.unknown');

	const params: Record<string, string | number> = { ...err.params };
	for (const name of ['field', 'other']) {
		const label = `field.${params[name]}`;
		if (name in params && t.has(label)) params[name] = t(label);
	}

	return t(key, params);
}

const [getTranslator, setTranslator, hasTranslator] = createContext<Translate>();

export function provideTranslator(translate: Translate) {
	setTranslator(translate);
}

/** The page's translator; outside the root layout (a component on its own),
 *  one over the base language. */
export function useTranslator(): Translate {
	return hasTranslator() ? getTranslator() : translator(() => base);
}

export function chooseLanguage(
	chosen: string | undefined,
	accepted: string | null,
	offered: string[],
	fallback: string
): string {
	const usable = (code: string | undefined | null) =>
		code && offered.includes(code) ? code : null;

	const wanted = usable(chosen);
	if (wanted) return wanted;

	for (const tag of parseAccepted(accepted)) {
		const exact = usable(tag);
		if (exact) return exact;

		const language = usable(tag.split('-')[0]);
		if (language) return language;
	}

	return usable(fallback) ?? offered[0] ?? BASE;
}

function parseAccepted(header: string | null): string[] {
	if (!header) return [];

	return header
		.split(',')
		.map((part) => {
			const [tag, ...rest] = part.trim().split(';');
			const quality = rest
				.map((piece) => piece.trim())
				.find((piece) => piece.startsWith('q='))
				?.slice(2);

			return { tag: tag.trim(), quality: quality === undefined ? 1 : Number(quality) || 0 };
		})
		.filter((entry) => entry.tag !== '' && entry.tag !== '*' && entry.quality > 0)
		.sort((a, b) => b.quality - a.quality)
		.map((entry) => entry.tag);
}
