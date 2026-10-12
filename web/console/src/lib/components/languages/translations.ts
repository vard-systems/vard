import { flatten } from './flatten';

function parameters(text: string): Set<string> {
	return new Set([...text.matchAll(/\{(\w+)\}/g)].map((match) => `{${match[1]}}`));
}

export function missingParameters(english: string, translation: string): string[] {
	const kept = parameters(translation);

	return [...parameters(english)].filter((name) => !kept.has(name));
}

export type ReadResult =
	{ ok: true; messages: Record<string, string>; skipped: number } | { ok: false };

export async function readTranslationFile(file: File, known: string[]): Promise<ReadResult> {
	let parsed: unknown;
	try {
		parsed = JSON.parse(await file.text());
	} catch {
		return { ok: false };
	}

	const flat = flatten(parsed);
	if (!flat) return { ok: false };

	const keys = new Set(known);
	const messages: Record<string, string> = {};
	let skipped = 0;

	for (const [key, value] of Object.entries(flat)) {
		if (key.startsWith('$')) continue;

		if (!keys.has(key)) {
			skipped++;
		} else if (value.trim() !== '') {
			messages[key] = value;
		}
	}

	return { ok: true, messages, skipped };
}

export function download(filename: string, text: string) {
	const url = URL.createObjectURL(new Blob([text], { type: 'application/json' }));

	const link = document.createElement('a');
	link.href = url;
	link.download = filename;
	link.click();

	URL.revokeObjectURL(url);
}

export const languageCode = /^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$/i;

export function namesOf(code: string): { name: string; native: string } | null {
	try {
		const [tag] = Intl.getCanonicalLocales(code);
		const name = new Intl.DisplayNames(['en'], { type: 'language', fallback: 'none' }).of(tag);
		const native = new Intl.DisplayNames([tag], { type: 'language', fallback: 'none' }).of(tag);

		if (!name) return null;

		return { name, native: native ? capitalise(native, tag) : name };
	} catch {
		return null;
	}
}

function capitalise(text: string, tag: string): string {
	return text.charAt(0).toLocaleUpperCase(tag) + text.slice(1);
}
