export type Messages = Record<string, string>;

export type Nested = { [key: string]: string | Nested };

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

export function nest(messages: Messages): Nested {
	const out: Nested = {};

	for (const [key, value] of Object.entries(messages)) {
		if (key.startsWith('$')) {
			out[key] = value;
			continue;
		}

		const parts = key.split('.');
		let node = out;

		for (const part of parts.slice(0, -1)) {
			const next = node[part];
			if (typeof next === 'string') break;
			node = (node[part] ??= {}) as Nested;
		}

		node[parts[parts.length - 1]] = value;
	}

	return out;
}

function isObject(value: unknown): value is Record<string, unknown> {
	return typeof value === 'object' && value !== null && !Array.isArray(value);
}
