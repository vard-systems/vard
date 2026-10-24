/** Hands `text` to the browser as a file to save. Nothing leaves the
    machine: the file is made here, from what the page already has. */
export function saveFile(name: string, text: string, type = 'application/json'): void {
	const url = URL.createObjectURL(new Blob([text], { type }));
	const link = Object.assign(document.createElement('a'), { href: url, download: name });
	link.click();
	URL.revokeObjectURL(url);
}

/** `value` as an indented JSON file. */
export function saveJson(name: string, value: unknown): void {
	saveFile(name, JSON.stringify(value, null, 2));
}

/** A file name that says what and when: `roles-2026-10-24.json`. */
export function datedName(subject: string, now = new Date()): string {
	return `${subject}-${now.toISOString().slice(0, 10)}.json`;
}
