export type Messages = Record<string, string>;

export function flatten(file: unknown): Messages | null {
	if (!isObject(file)) return null;

	const out: Messages = {};

	const walk = (node: Record<string, unknown>, prefix: string): boolean => {
		for (const [key, value] of Object.entries(node)) {
			const full = prefix ? `${prefix}.${key}` : key;

			if (typeof value === 'string') {
				if (full in out) return false;
				out[full] = value;
			} else if (isObject(value)) {
				if (!walk(value, full)) return false;
			} else {
				return false;
			}
		}

		return true;
	};

	return walk(file, '') ? out : null;
}

function isObject(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}
