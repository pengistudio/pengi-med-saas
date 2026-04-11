/**
 * Genera un ID personalizado para tareas basado en el nombre de la empresa
 * Ejemplos:
 * - "Pengi Studio" → "PEN-1" (3 letras iniciales)
 * - "Acme Corp Inc" → "ACI-1" (iniciales de 3+ palabras)
 * - "ABC" → "ABC-1" (nombre corto ≤4 letras)
 * - "John" → "JOHN-1" (nombre muy corto, usa todas las letras)
 */
export function generateTaskId(
	companyName: string,
	taskNumber: number,
): string {
	if (!companyName || companyName.trim() === "") {
		return `GEN-${taskNumber}`;
	}

	const trimmed = companyName.trim();
	const words = trimmed.split(/\s+/).filter((w) => w.length > 0);

	let prefix: string;

	if (words.length === 1) {
		// Nombre sin espacios
		if (trimmed.length <= 4) {
			// Muy corto: usar todas las letras
			prefix = trimmed.toUpperCase();
		} else {
			// Usar las 3 primeras letras
			prefix = trimmed.substring(0, 3).toUpperCase();
		}
	} else if (words.length >= 3) {
		// 3 o más palabras: usar iniciales
		prefix = words
			.slice(0, 3)
			.map((w) => w.charAt(0).toUpperCase())
			.join("");
	} else {
		// 2 palabras: usar primeras letras (hasta 3)
		prefix = words
			.map((w) => w.charAt(0).toUpperCase())
			.join("")
			.substring(0, 3);
	}

	return `${prefix}-${taskNumber}`;
}
