import type { AppText } from "@pengi/shared";

/**
 * The WhatsApp text for an exam order, in the interface language (mirrors
 * buildPrescriptionWhatsAppMessage in lib/utils.ts).
 */
export function buildExamOrderWhatsAppMessage(
	params: {
		code: string;
		patientName: string;
		doctorName?: string;
		date: string;
		urgent?: boolean;
		items: Array<{ name: string; indications?: string }>;
		destinationLab?: string;
		notes?: string;
	},
	textGet: AppText["textGet"],
): string {
	const lines: string[] = [];
	lines.push(
		`🧪 *${textGet("clinical.exam_orders.whatsapp.title", { code: params.code })}*`,
	);
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
	if (params.urgent)
		lines.push(`⚠️ *${textGet("clinical.exam_orders.whatsapp.urgent")}*`);
	if (params.items.length > 0) {
		lines.push(`\n📋 *${textGet("clinical.exam_orders.whatsapp.exams")}*`);
		for (const item of params.items) {
			lines.push(`• ${item.name}`);
			if (item.indications) lines.push(`  ${item.indications}`);
		}
	}
	if (params.destinationLab)
		lines.push(
			`\n🏥 ${textGet("clinical.exam_orders.whatsapp.lab", { lab: params.destinationLab })}`,
		);
	if (params.notes)
		lines.push(
			`\n📝 *${textGet("clinical.exam_orders.whatsapp.notes")}*\n${params.notes}`,
		);
	return lines.join("\n");
}
