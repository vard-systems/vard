export const LIST_PAGE_SIZE = 50;

type OffsetPage = { total: number; offset: number };

export function nextOffset(page: OffsetPage, count: number): number | undefined {
	const next = page.offset + count;

	return count > 0 && next < page.total ? next : undefined;
}

/**
 * Rows of every page loaded so far, deduplicated by id (offset pages can repeat a row after an
 * insert or delete).
 */
export function uniqueById<Row extends { id: string }>(rows: Row[]): Row[] {
	const seen = new Set<string>();

	return rows.filter((row) => !seen.has(row.id) && (seen.add(row.id), true));
}

/**
 * How many rows a fully-loaded list shows at first. The filter arguments are unused but read,
 * so a `$derived` of this resets to one page when the filter changes.
 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
export function firstPage(...filter: unknown[]): number {
	return LIST_PAGE_SIZE;
}
