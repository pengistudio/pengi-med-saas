import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

/**
 * Le da formato a la fecha según lo que se especifica
 * @param date - Fecha
 * @param options  - opciones en formato Intl.DateTimeFormatOptions (opcional)
 * @returns `string` con el formato de la fecha
 */
export function generateWhatsAppLink(
	phoneNumber: string,
	message?: string,
): string {
	let normalized = phoneNumber.trim().replace(/[^0-9+]/g, "");
	if (normalized.startsWith("0")) normalized = `593${normalized.slice(1)}`;
	if (normalized.startsWith("+")) normalized = normalized.slice(1);
	const base = `https://wa.me/${normalized}`;
	return message ? `${base}?text=${encodeURIComponent(message)}` : base;
}

export function buildPrescriptionWhatsAppMessage(params: {
	patientName: string;
	doctorName?: string;
	date: string;
	items?: Array<{
		medication: string;
		dose: string;
		frequency: string;
		duration: string;
		notes?: string;
	}>;
	indications?: string;
}): string {
	const lines: string[] = [];
	lines.push(`🏥 *Receta Médica*`);
	lines.push(`👤 Paciente: ${params.patientName}`);
	if (params.doctorName) lines.push(`👨‍⚕️ Médico: ${params.doctorName}`);
	lines.push(`📅 Fecha: ${params.date}`);
	if (params.items && params.items.length > 0) {
		lines.push(`\n💊 *Medicamentos:*`);
		for (const item of params.items) {
			lines.push(`• *${item.medication}*`);
			lines.push(
				`  Dosis: ${item.dose} | Frecuencia: ${item.frequency} | Duración: ${item.duration}`,
			);
			if (item.notes) lines.push(`  Notas: ${item.notes}`);
		}
	}
	if (params.indications) {
		lines.push(`\n📋 *Indicaciones:*\n${params.indications}`);
	}
	return lines.join("\n");
}

export function dateParser(
	date: Date | string,
	options?: Intl.DateTimeFormatOptions,
) {
	let transformDate = new Date();
	if (typeof date === "string") transformDate = new Date(date);
	else {
		transformDate = date;
	}
	if (options)
		return Intl.DateTimeFormat("es-EC", options).format(transformDate);
	return Intl.DateTimeFormat("es-EC", { dateStyle: "medium" }).format(
		transformDate,
	);
}
