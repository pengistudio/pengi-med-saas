import { useMessageStore, useText } from "@pengi/shared";
import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { buildPrescriptionWhatsAppMessage } from "@/lib/utils";
import en from "../../../api/i18n/messages/messages_en.json";
import es from "../../../api/i18n/messages/messages_es.json";

type Catalog = Array<{ key: string; value: string }>;

const toMessages = (catalog: Catalog) =>
	Object.fromEntries(catalog.map(({ key, value }) => [key, value]));

const textGetFor = (lang: "es" | "en", catalog: Catalog) => {
	useMessageStore.setState({ lang, messages: toMessages(catalog) });
	return renderHook(() => useText()).result.current.textGet;
};

const params = {
	patientName: "Juan Pérez",
	doctorName: "Dra. Ana Ruiz",
	date: "5 mar 2026",
	items: [
		{
			medication: "Amoxicilina 500mg",
			dose: "1 cápsula",
			frequency: "cada 8 horas",
			duration: "7 días",
			notes: "Tomar con comida",
		},
		{
			medication: "Paracetamol",
			dose: "1 tableta",
			frequency: "cada 6 horas",
			duration: "3 días",
		},
	],
	indications: "Reposo e hidratación",
};

describe("buildPrescriptionWhatsAppMessage", () => {
	it("builds the Spanish message", () => {
		expect(buildPrescriptionWhatsAppMessage(params, textGetFor("es", es))).toBe(
			[
				"🏥 *Receta Médica*",
				"👤 Paciente: Juan Pérez",
				"👨‍⚕️ Médico: Dra. Ana Ruiz",
				"📅 Fecha: 5 mar 2026",
				"\n💊 *Medicamentos:*",
				"• *Amoxicilina 500mg*",
				"  Dosis: 1 cápsula | Frecuencia: cada 8 horas | Duración: 7 días",
				"  Notas: Tomar con comida",
				"• *Paracetamol*",
				"  Dosis: 1 tableta | Frecuencia: cada 6 horas | Duración: 3 días",
				"\n📋 *Indicaciones:*\nReposo e hidratación",
			].join("\n"),
		);
	});

	it("builds the English message", () => {
		const message = buildPrescriptionWhatsAppMessage(
			params,
			textGetFor("en", en),
		);
		expect(message).toContain("🏥 *Medical Prescription*");
		expect(message).toContain("👤 Patient: Juan Pérez");
		expect(message).toContain(
			"  Dose: 1 cápsula | Frequency: cada 8 horas | Duration: 7 días",
		);
		expect(message).not.toContain("*clinical.");
	});

	it("omits the doctor, medications and indications when absent", () => {
		expect(
			buildPrescriptionWhatsAppMessage(
				{ patientName: "Juan Pérez", date: "5 mar 2026", items: [] },
				textGetFor("es", es),
			),
		).toBe("🏥 *Receta Médica*\n👤 Paciente: Juan Pérez\n📅 Fecha: 5 mar 2026");
	});
});
