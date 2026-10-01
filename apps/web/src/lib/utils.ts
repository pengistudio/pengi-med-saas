import type { AppText } from "@pengi/shared";
import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
	return twMerge(clsx(inputs));
}

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

/**
 * The WhatsApp text for a prescription, in the interface language.
 * `params.date` is already formatted by the caller (`formatDate`).
 */
export function buildPrescriptionWhatsAppMessage(
	params: {
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
	},
	textGet: AppText["textGet"],
): string {
	const lines: string[] = [];
	lines.push(`🏥 *${textGet("clinical.prescription.whatsapp_message.title")}*`);
	lines.push(
		`👤 ${textGet("clinical.prescription.whatsapp_message.patient", { name: params.patientName })}`,
	);
	if (params.doctorName)
		lines.push(
			`👨‍⚕️ ${textGet("clinical.prescription.whatsapp_message.doctor", { name: params.doctorName })}`,
		);
	lines.push(
		`📅 ${textGet("clinical.prescription.whatsapp_message.date", { date: params.date })}`,
	);
	if (params.items && params.items.length > 0) {
		lines.push(
			`\n💊 *${textGet("clinical.prescription.whatsapp_message.medications")}*`,
		);
		for (const item of params.items) {
			lines.push(`• *${item.medication}*`);
			lines.push(
				`  ${textGet("clinical.prescription.whatsapp_message.item_detail", {
					dose: item.dose,
					frequency: item.frequency,
					duration: item.duration,
				})}`,
			);
			if (item.notes)
				lines.push(
					`  ${textGet("clinical.prescription.whatsapp_message.item_notes", { notes: item.notes })}`,
				);
		}
	}
	if (params.indications) {
		lines.push(
			`\n📋 *${textGet("clinical.prescription.whatsapp_message.indications")}*\n${params.indications}`,
		);
	}
	return lines.join("\n");
}
