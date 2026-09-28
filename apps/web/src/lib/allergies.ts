/**
 * A patient's `allergies` as a list. The API stores them as a JSON array of
 * strings ("[\"penicilina\"]"), but older records and the medical record form
 * hold plain comma-separated text. An empty array, an empty string or only
 * blanks is "no allergies".
 */
export function parseAllergies(allergies?: string | null): string[] {
	const raw = allergies?.trim();
	if (!raw) return [];

	if (raw.startsWith("[")) {
		try {
			const parsed: unknown = JSON.parse(raw);
			if (Array.isArray(parsed)) return clean(parsed.map(String));
		} catch {
			// Not JSON after all: fall back to comma-separated text.
		}
	}

	return clean(raw.split(","));
}

function clean(items: string[]): string[] {
	const seen = new Set<string>();
	for (const item of items) {
		const value = item.trim();
		if (value) seen.add(value);
	}
	return [...seen];
}
