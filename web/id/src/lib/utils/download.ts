/** Hands the reader a text file made here, without a round trip to the server. */
export function saveText(name: string, text: string): void {
	const url = URL.createObjectURL(new Blob([text], { type: 'text/plain;charset=utf-8' }));
	const link = Object.assign(document.createElement('a'), { href: url, download: name });
	link.click();
	URL.revokeObjectURL(url);
}
