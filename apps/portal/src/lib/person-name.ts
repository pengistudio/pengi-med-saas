// Spanish particles stay lowercase inside a name: "Juan de la Cruz".
const PARTICLES = new Set(["de", "del", "la", "las", "los", "y", "e"]);

/**
 * Names are often typed in capitals at the front desk ("WALTER SANTACRUZ").
 * Shown to a patient they read as shouting, so they get name case; names
 * that are already mixed case are left as typed.
 */
export function nameCase(name: string): string {
	if (name !== name.toUpperCase()) return name;
	return name
		.toLowerCase()
		.split(/(\s+|-)/)
		.map((part, i) =>
			i > 0 && PARTICLES.has(part)
				? part
				: part.charAt(0).toUpperCase() + part.slice(1),
		)
		.join("");
}
